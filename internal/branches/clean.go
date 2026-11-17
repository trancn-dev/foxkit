package branches

import (
	"context"
	"errors"
	"path"
	"sort"
	"strings"

	"foxkit/internal/git"
)

type Candidate struct {
	Name   string `json:"name"`
	Reason string `json:"reason"` // merged | gone
	Last   string `json:"last"`
}

type Plan struct {
	Applied bool        `json:"applied"`
	Base    string      `json:"base"`
	Delete  []Candidate `json:"delete"`
	Kept    []string    `json:"kept"`
	Failed  []string    `json:"failed"`
}

var (
	ErrNotRepo      = errors.New("không phải repo git")
	ErrBaseNotFound = errors.New("không đoán được branch gốc")
)

var protectedNames = []string{"main", "master", "develop", "dev", "staging", "production", "release/*"}

func lines(b []byte) []string {
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(strings.TrimRight(l, "\r")); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func guessBase(ctx context.Context, dir string) (string, error) {
	if out, err := git.Run(ctx, dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if b := strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/"); b != "" {
			return b, nil
		}
	}
	for _, b := range []string{"main", "master", "develop"} {
		if _, err := git.Run(ctx, dir, "show-ref", "--verify", "--quiet", "refs/heads/"+b); err == nil {
			return b, nil
		}
	}
	return "", ErrBaseNotFound
}

func isProtected(name string, patterns []string) bool {
	for _, p := range patterns {
		if ok, _ := path.Match(p, name); ok { // pattern cố định/ do người dùng nhập; sai cú pháp thì coi như không khớp
			return true
		}
	}
	return false
}

// Build lập kế hoạch xóa: branch đã merge vào base (và tùy chọn: upstream đã mất). Không đụng branch bảo vệ/đang đứng/base.
func Build(ctx context.Context, dir, base string, gone bool, keep []string) (Plan, error) {
	p := Plan{Delete: []Candidate{}, Kept: []string{}, Failed: []string{}}
	if out, err := git.Run(ctx, dir, "rev-parse", "--is-inside-work-tree"); err != nil || strings.TrimSpace(string(out)) != "true" {
		if errors.Is(err, git.ErrNotFound) {
			return p, err
		}
		return p, ErrNotRepo
	}
	if base == "" {
		var err error
		if base, err = guessBase(ctx, dir); err != nil {
			return p, err
		}
	} else if _, err := git.Run(ctx, dir, "rev-parse", "--verify", "--quiet", base); err != nil {
		return p, ErrBaseNotFound
	}
	p.Base = base

	current := ""
	if out, err := git.Run(ctx, dir, "branch", "--show-current"); err == nil {
		current = strings.TrimSpace(string(out))
	}
	patterns := append(append([]string{base, current}, protectedNames...), keep...)

	all, err := git.Run(ctx, dir, "for-each-ref", "refs/heads", "--format=%(refname:short)|%(upstream:track)")
	if err != nil {
		return p, err
	}
	goneSet := map[string]bool{}
	var locals []string
	for _, l := range lines(all) {
		name, track, _ := strings.Cut(l, "|")
		locals = append(locals, name)
		if strings.Contains(track, "[gone]") {
			goneSet[name] = true
		}
	}
	merged := map[string]bool{}
	if out, err := git.Run(ctx, dir, "branch", "--merged", base, "--format=%(refname:short)"); err == nil {
		for _, l := range lines(out) {
			merged[l] = true
		}
	}
	for _, name := range locals {
		switch {
		case isProtected(name, patterns):
			p.Kept = append(p.Kept, name)
		case merged[name]:
			p.Delete = append(p.Delete, Candidate{Name: name, Reason: "merged", Last: lastDate(ctx, dir, name)})
		case gone && goneSet[name]:
			p.Delete = append(p.Delete, Candidate{Name: name, Reason: "gone", Last: lastDate(ctx, dir, name)})
		}
	}
	sort.Strings(p.Kept)
	return p, nil
}

func lastDate(ctx context.Context, dir, branch string) string {
	out, err := git.Run(ctx, dir, "log", "-1", "--format=%cs", branch)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Apply xóa: merged → `branch -d` (git tự từ chối nếu chưa merge), gone → `-D` (squash merge git không nhận là đã merge).
// Lỗi từng branch vào Failed, tiếp tục.
func Apply(ctx context.Context, dir string, p *Plan) error {
	for _, c := range p.Delete {
		flag := "-d"
		if c.Reason == "gone" {
			flag = "-D"
		}
		if _, err := git.Run(ctx, dir, "branch", flag, c.Name); err != nil {
			p.Failed = append(p.Failed, c.Name)
		}
	}
	p.Applied = true
	return ctx.Err()
}
