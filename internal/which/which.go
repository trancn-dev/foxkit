package which

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"foxkit/internal/winexec"
	"foxkit/internal/winpath"
)

type Match struct {
	Path    string `json:"path"`
	Dir     string `json:"dir"`
	Active  bool   `json:"active"`
	Version string `json:"version,omitempty"`
}

type Result struct {
	Name    string  `json:"name"`
	Matches []Match `json:"matches"`
}

const defaultExt = ".COM;.EXE;.BAT;.CMD"

// Find liệt kê mọi bản của name theo thứ tự ưu tiên của cmd.exe: cwd trước, rồi từng mục PATH.
// Nhận pathEnv/pathExt/cwd làm tham số để test được mà không đụng biến môi trường thật.
func Find(name, pathEnv, pathExt, cwd string) []Match {
	if strings.TrimSpace(pathExt) == "" {
		pathExt = defaultExt
	}
	var exts []string
	for _, e := range strings.Split(strings.ToLower(pathExt), ";") {
		if e = strings.TrimSpace(e); e != "" {
			exts = append(exts, e)
		}
	}
	dirs := []string{}
	if cwd != "" {
		dirs = append(dirs, cwd)
	}
	for _, d := range strings.Split(pathEnv, ";") {
		if d = strings.Trim(strings.TrimSpace(winpath.Expand(d)), `"`); d != "" {
			dirs = append(dirs, d)
		}
	}
	lname := strings.ToLower(name)
	hasExt := false
	for _, e := range exts {
		if strings.HasSuffix(lname, e) {
			hasExt = true
		}
	}
	want := []string{lname}
	if !hasExt {
		want = want[:0]
		for _, e := range exts {
			want = append(want, lname+e)
		}
	}
	matches := []Match{}
	seen := map[string]bool{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir) // đọc tên thật để giữ đúng chữ hoa/thường
		if err != nil {
			continue
		}
		actual := map[string]os.DirEntry{}
		for _, e := range entries {
			actual[strings.ToLower(e.Name())] = e
		}
		for _, w := range want {
			e, ok := actual[w]
			if !ok || e.IsDir() {
				continue
			}
			p := filepath.Join(dir, e.Name())
			key := strings.ToLower(filepath.Clean(p))
			if seen[key] {
				continue
			}
			seen[key] = true
			matches = append(matches, Match{Path: p, Dir: filepath.Clean(dir)})
		}
	}
	if len(matches) > 0 {
		matches[0].Active = true
	}
	return matches
}

// Versions chạy `<exe> --version` (php dùng -v) song song, timeout 3s, ghi dòng đầu (≤80 ký tự) vào Match.Version.
func Versions(ctx context.Context, ms []Match) {
	var wg sync.WaitGroup
	for i := range ms {
		wg.Add(1)
		go func(m *Match) {
			defer wg.Done()
			flag := "--version"
			base := strings.ToLower(strings.TrimSuffix(filepath.Base(m.Path), filepath.Ext(m.Path)))
			if base == "php" {
				flag = "-v"
			}
			name, args := m.Path, []string{flag}
			if ext := strings.ToLower(filepath.Ext(m.Path)); ext == ".bat" || ext == ".cmd" {
				name, args = "cmd", []string{"/c", m.Path, flag}
			}
			out, _ := winexec.RunCombined(ctx, 3*time.Second, name, args...) // lỗi/timeout: để trống, không phải lỗi lệnh
			m.Version = firstLine(string(out))
		}(&ms[i])
	}
	wg.Wait()
}

func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if r := []rune(l); len(r) > 80 {
				return string(r[:80])
			}
			return l
		}
	}
	return ""
}
