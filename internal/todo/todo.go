package todo

import (
	"bufio"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"foxkit/internal/fsutil"
	"foxkit/internal/winexec"
)

type Item struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Tag    string `json:"tag"`
	Text   string `json:"text"`
	Author string `json:"author,omitempty"`
}

type Report struct {
	Dir    string         `json:"dir"`
	Items  []Item         `json:"items"`
	Counts map[string]int `json:"counts"`
	Files  int            `json:"files"`
}

type Options struct {
	Dir    string
	Tags   []string
	Exts   []string
	Author bool
}

var DefaultTags = []string{"TODO", "FIXME", "HACK", "XXX"}
var DefaultExts = []string{"php", "js", "ts", "jsx", "tsx", "vue", "go", "css", "scss", "blade.php", "py", "sql", "md"}

// extraSkip: thư mục (so theo đường dẫn tương đối, dấu /) cần bỏ thêm ngoài DefaultSkipDirs.
var extraSkip = []string{"public/build", "public/vendor", "bootstrap/cache", "coverage"}

const maxFileBytes = 2 << 20

var tagNameRe = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func ValidTags(tags []string) bool {
	if len(tags) == 0 {
		return false
	}
	for _, t := range tags {
		if !tagNameRe.MatchString(t) {
			return false
		}
	}
	return true
}

// Matcher biên dịch regex một lần cho cả lần quét.
func Matcher(tags []string) *regexp.Regexp {
	q := make([]string, len(tags))
	for i, t := range tags {
		q[i] = regexp.QuoteMeta(t)
	}
	// tag phải nằm sau dấu comment trên cùng dòng; "(tên):" ngay sau tag là ghi chú tác giả, bỏ đi
	return regexp.MustCompile(`(?://|#|/\*|\*|<!--|\{\{--)[^\n]*?\b(` + strings.Join(q, "|") + `)\b(?:\([^)]*\))?:?\s*(.*)`)
}

var closers = []string{"*/", "-->", "--}}"}

// ParseLine trả (tag, text, ok) cho một dòng mã nguồn.
func ParseLine(re *regexp.Regexp, line string) (string, string, bool) {
	m := re.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	text := strings.TrimSpace(m[2])
	for changed := true; changed; {
		changed = false
		for _, c := range closers {
			if strings.HasSuffix(text, c) {
				text = strings.TrimSpace(strings.TrimSuffix(text, c))
				changed = true
			}
		}
	}
	if r := []rune(text); len(r) > 200 {
		text = string(r[:200])
	}
	return m[1], text, true
}

func skipFile(name string) bool {
	l := strings.ToLower(name)
	return strings.HasSuffix(l, ".min.js") || strings.HasSuffix(l, ".min.css")
}

func extOK(name string, exts []string) bool {
	l := strings.ToLower(name)
	for _, e := range exts {
		if strings.HasSuffix(l, "."+strings.ToLower(strings.TrimPrefix(e, "."))) {
			return true
		}
	}
	return false
}

func scanFile(re *regexp.Regexp, path, rel string) []Item {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var items []Item
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for no := 1; sc.Scan(); no++ {
		if tag, text, ok := ParseLine(re, sc.Text()); ok {
			items = append(items, Item{File: rel, Line: no, Tag: tag, Text: text})
		}
	}
	// dòng dài hơn 1MB làm Scanner dừng (ErrTooLong): giữ những gì đã đọc được, không coi là lỗi
	return items
}

func Run(ctx context.Context, opt Options) (Report, error) {
	if len(opt.Tags) == 0 {
		opt.Tags = DefaultTags
	}
	if len(opt.Exts) == 0 {
		opt.Exts = DefaultExts
	}
	re := Matcher(opt.Tags)
	rep := Report{Dir: opt.Dir, Items: []Item{}, Counts: map[string]int{}}

	type job struct{ path, rel string }
	jobs := make(chan job)
	results := make(chan []Item)
	var wg sync.WaitGroup
	for i := 0; i < runtime.NumCPU(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				results <- scanFile(re, j.path, j.rel)
			}
		}()
	}
	var walkErr error
	go func() {
		defer close(jobs)
		walkErr = filepath.WalkDir(opt.Dir, func(p string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if fsutil.Skippable(d) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			rel, _ := filepath.Rel(opt.Dir, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if p != opt.Dir && (fsutil.IsSkipDir(d.Name(), fsutil.DefaultSkipDirs) || has(extraSkip, rel) || d.Name() == "coverage") {
					return fs.SkipDir
				}
				return nil
			}
			if skipFile(d.Name()) || !extOK(d.Name(), opt.Exts) {
				return nil
			}
			if info, err := d.Info(); err != nil || info.Size() > maxFileBytes {
				return nil
			}
			select {
			case jobs <- job{p, rel}:
			case <-ctx.Done():
				return ctx.Err()
			}
			return nil
		})
	}()
	go func() { wg.Wait(); close(results) }()
	for items := range results {
		rep.Files++
		rep.Items = append(rep.Items, items...)
	}
	if walkErr != nil {
		return rep, walkErr
	}
	sort.Slice(rep.Items, func(i, j int) bool {
		if rep.Items[i].File != rep.Items[j].File {
			return rep.Items[i].File < rep.Items[j].File
		}
		return rep.Items[i].Line < rep.Items[j].Line
	})
	if opt.Author {
		addAuthors(ctx, opt.Dir, rep.Items)
	}
	for _, it := range rep.Items {
		rep.Counts[it.Tag]++
	}
	return rep, nil
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// addAuthors: git blame từng dòng (tối đa 8 song song). Không phải repo git / không có git → bỏ qua im lặng.
func addAuthors(ctx context.Context, dir string, items []Item) {
	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	for i := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(it *Item) {
			defer wg.Done()
			defer func() { <-sem }()
			arg := strconv.Itoa(it.Line) + "," + strconv.Itoa(it.Line)
			out, err := winexec.Run(ctx, 10*time.Second, "git", "-C", dir, "blame", "-L", arg, "--porcelain", "--", filepath.FromSlash(it.File))
			if err != nil {
				return
			}
			for _, l := range strings.Split(string(out), "\n") {
				if a, ok := strings.CutPrefix(l, "author "); ok {
					it.Author = strings.TrimSpace(a)
					return
				}
			}
		}(&items[i])
	}
	wg.Wait()
}
