package prune

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"foxkit/internal/fsutil"
)

type Item struct {
	Path         string    `json:"path"`
	Project      string    `json:"project"`
	Target       string    `json:"target"`
	Bytes        int64     `json:"bytes"`
	LastActivity time.Time `json:"lastActivity"`
	Deleted      bool      `json:"deleted"`
	Error        string    `json:"error,omitempty"`
}

type Report struct {
	Applied    bool   `json:"applied"`
	Items      []Item `json:"items"`
	TotalBytes int64  `json:"totalBytes"`
	FreedBytes int64  `json:"freedBytes"`
	Skipped    int    `json:"skipped"`
}

type Options struct {
	Dir     string
	Targets []string
	Older   time.Duration // 0 = mọi dự án
	Depth   int
	Apply   bool
	Now     time.Time
}

// SafeTargets: chỉ những thư mục này được phép xóa.
var SafeTargets = []string{"node_modules", "vendor", ".next", ".nuxt", "dist", ".turbo", ".cache"}

// marker: file ở thư mục cha chứng tỏ target đó thuộc một dự án (tránh xóa nhầm vendor của Go).
func marker(target string) string {
	if target == "vendor" {
		return "composer.json"
	}
	return "package.json"
}

// activityFiles: file/thư mục cấp đầu của dự án dùng để đo lần đụng tới gần nhất (không đệ quy, không tính target).
var activityFiles = []string{
	"package.json", "composer.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "composer.lock", ".env",
	"src", "app", "resources", ".git/HEAD", ".git/index",
}

func lastActivity(project string) time.Time {
	var latest time.Time
	for _, f := range activityFiles {
		if st, err := os.Stat(filepath.Join(project, filepath.FromSlash(f))); err == nil && st.ModTime().After(latest) {
			latest = st.ModTime()
		}
	}
	if latest.IsZero() {
		if st, err := os.Stat(project); err == nil {
			return st.ModTime()
		}
	}
	return latest
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Run quét tìm target của các dự án cũ; có Apply thì xóa. Lỗi từng mục nằm trong Item.Error, không làm hỏng cả report.
func Run(ctx context.Context, opt Options) (Report, error) {
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	rep := Report{Applied: opt.Apply, Items: []Item{}}
	activity := map[string]time.Time{}
	skippedProjects := map[string]bool{}
	var items []Item

	root := filepath.Clean(opt.Dir)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.IsDir() || p == root {
			return nil
		}
		if fsutil.Skippable(d) || d.Name() == ".git" {
			return fs.SkipDir
		}
		rel, _ := filepath.Rel(root, p)
		if strings.Count(rel, string(os.PathSeparator))+1 > opt.Depth {
			return fs.SkipDir
		}
		name := d.Name()
		if !has(SafeTargets, name) {
			return nil
		}
		// đã là target thì không đi vào trong, dù có ghi nhận hay không (tiết kiệm thời gian quét lớn nhất)
		if !has(opt.Targets, name) {
			return fs.SkipDir
		}
		project := filepath.Dir(p)
		if !fsutil.Exists(filepath.Join(project, marker(name))) {
			return fs.SkipDir
		}
		la, ok := activity[project]
		if !ok {
			la = lastActivity(project)
			activity[project] = la
		}
		if opt.Older > 0 && opt.Now.Sub(la) < opt.Older {
			skippedProjects[project] = true
			return fs.SkipDir
		}
		items = append(items, Item{Path: p, Project: project, Target: name, LastActivity: la})
		return fs.SkipDir
	})
	if err != nil {
		return rep, err
	}
	rep.Skipped = len(skippedProjects)

	// size song song (semaphore)
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU()*2)
	for i := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			st, _ := fsutil.DirSize(ctx, items[i].Path) // lỗi duy nhất là ctx hủy, kiểm ở dưới
			items[i].Bytes = st.Bytes
		}(i)
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return rep, err
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Bytes > items[b].Bytes })
	for i := range items {
		rep.TotalBytes += items[i].Bytes
		if opt.Apply {
			if err := removeAllForce(items[i].Path); err != nil {
				items[i].Error = err.Error()
			} else {
				items[i].Deleted = true
				rep.FreedBytes += items[i].Bytes
			}
		}
	}
	rep.Items = append(rep.Items, items...)
	return rep, nil
}

// removeAllForce: RemoveAll; gặp file read-only (access denied) thì bỏ cờ read-only từng mục rồi thử lại một lần.
func removeAllForce(path string) error {
	err := os.RemoveAll(path)
	if err == nil {
		return nil
	}
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, werr error) error { // chỉ cố mở khóa quyền; lỗi từng mục bỏ qua
		if werr == nil {
			_ = os.Chmod(p, 0o666)
		}
		return nil
	})
	return os.RemoveAll(path)
}
