package repos

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"foxkit/internal/git"
)

type Repo struct {
	Path string `json:"path"`
	Name string `json:"name"`
	git.Status
	LastCommit string `json:"lastCommit"`
	Error      string `json:"error,omitempty"`
}

// Attention: repo có vấn đề (dirty, chưa push, bị tụt, chưa có upstream, lỗi).
func (r Repo) Attention() bool {
	return r.Error != "" || r.Staged+r.Modified+r.Untracked+r.Conflicts > 0 || r.Ahead > 0 || r.Behind > 0 || r.Upstream == ""
}

// find tìm thư mục chứa `.git` (thư mục hoặc file: worktree/submodule), tới độ sâu depth, không quét repo lồng.
func find(root string, depth int) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error { // lỗi từng thư mục (quyền) bỏ qua
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == "node_modules" || d.Name() == "vendor" {
			return fs.SkipDir
		}
		rel, _ := filepath.Rel(root, p)
		if rel != "." && strings.Count(rel, string(os.PathSeparator))+1 > depth {
			return fs.SkipDir
		}
		if _, err := os.Stat(filepath.Join(p, ".git")); err == nil {
			out = append(out, p)
			return fs.SkipDir
		}
		return nil
	})
	return out
}

func inspect(ctx context.Context, path string, fetch bool) Repo {
	r := Repo{Path: path, Name: filepath.Base(path)}
	if fetch {
		fctx, cancel := context.WithTimeout(ctx, 20*time.Second)
		if _, err := git.Run(fctx, path, "fetch", "--quiet", "--no-tags"); err != nil {
			r.Error = "fetch lỗi (không có mạng hoặc cần đăng nhập)"
		}
		cancel()
	}
	out, err := git.Run(ctx, path, "status", "--porcelain=v2", "--branch")
	if err != nil {
		r.Error = firstLine(err.Error())
		return r
	}
	st, err := git.ParseStatusV2(out)
	if err != nil {
		r.Error = firstLine(err.Error())
		return r
	}
	r.Status = st
	if out, err := git.Run(ctx, path, "stash", "list"); err == nil {
		if s := strings.TrimSpace(string(out)); s != "" {
			r.Stashes = len(strings.Split(s, "\n"))
		}
	}
	if out, err := git.Run(ctx, path, "log", "-1", "--format=%cI"); err == nil { // repo chưa có commit: lỗi, để trống
		r.LastCommit = strings.TrimSpace(string(out))
	}
	return r
}

// Scan quét dir, tối đa 8 repo song song. Repo cần chú ý xếp trước, rồi theo tên.
func Scan(ctx context.Context, dir string, depth int, fetch bool) ([]Repo, error) {
	if !git.Available() {
		return nil, git.ErrNotFound
	}
	paths := find(dir, depth)
	repos := make([]Repo, len(paths))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, p := range paths {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, p string) {
			defer wg.Done()
			defer func() { <-sem }()
			repos[i] = inspect(ctx, p, fetch)
		}(i, p)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(repos, func(a, b int) bool {
		if repos[a].Attention() != repos[b].Attention() {
			return repos[a].Attention()
		}
		return strings.ToLower(repos[a].Name) < strings.ToLower(repos[b].Name)
	})
	return repos, nil
}

// firstLine: lỗi git nhiều dòng (vd "dubious ownership") chỉ lấy dòng đầu cho gọn.
func firstLine(s string) string { return strings.SplitN(s, "
", 2)[0] }
