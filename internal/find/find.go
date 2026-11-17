package find

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"foxkit/internal/fsutil"
	"foxkit/internal/textcase"
)

type Hit struct {
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	IsDir    bool      `json:"isDir"`
	Bytes    int64     `json:"bytes"`
	Modified time.Time `json:"modified"`
	Line     int       `json:"line,omitempty"`
	Snippet  string    `json:"snippet,omitempty"`
}

type Result struct {
	Dir       string `json:"dir"`
	Hits      []Hit  `json:"hits"`
	Truncated bool   `json:"truncated"`
	Scanned   int    `json:"scanned"`
}

type Options struct {
	Dir        string
	Pattern    string
	Exts       []string
	Type       string // f | d | a
	Larger     *int64
	Smaller    *int64
	Newer      time.Duration // sửa trong vòng; 0 = không lọc
	Older      time.Duration // cũ hơn; 0 = không lọc
	Content    string
	IgnoreCase bool
	Max        int
	Hidden     bool
	Now        time.Time
}

var errStop = errors.New("đủ kết quả")

const maxContentBytes = 20 << 20

// Normalize: bỏ dấu tiếng Việt + chữ thường, để "hop dong" khớp "Hợp Đồng".
func Normalize(s string) string { return strings.ToLower(textcase.RemoveDiacritics(s)) }

func isGlob(p string) bool { return strings.ContainsAny(p, "*?") }

// ValidPattern: glob có hợp lệ không (vd `[` không đóng là sai).
func ValidPattern(p string) bool {
	_, err := filepath.Match(Normalize(p), "")
	return err == nil
}

func matchName(pattern, name string) bool {
	if pattern == "" {
		return true
	}
	p, n := Normalize(pattern), Normalize(name)
	if isGlob(p) {
		ok, _ := filepath.Match(p, n) // pattern đã kiểm bằng ValidPattern trước khi chạy
		return ok
	}
	return strings.Contains(n, p)
}

func Run(ctx context.Context, opt Options) (Result, error) {
	if opt.Max < 1 {
		opt.Max = 200
	}
	if opt.Type == "" {
		opt.Type = "f"
	}
	if opt.Now.IsZero() {
		opt.Now = time.Now()
	}
	exts := map[string]bool{}
	for _, e := range opt.Exts {
		exts[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e), "."))] = true
	}
	res := Result{Dir: opt.Dir, Hits: []Hit{}}

	parent := ctx
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	jobs := make(chan Hit)
	results := make(chan Hit)
	var wg sync.WaitGroup
	if opt.Content != "" {
		for i := 0; i < runtime.NumCPU(); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for h := range jobs {
					if line, snip, ok := searchContent(h.Path, opt.Content, opt.IgnoreCase); ok {
						h.Line, h.Snippet = line, snip
						select {
						case results <- h:
						case <-ctx.Done():
							return
						}
					}
				}
			}()
		}
		go func() { wg.Wait(); close(results) }()
	}

	walk := func() error {
		return filepath.WalkDir(opt.Dir, func(p string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return errStop
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
			isRoot := p == opt.Dir
			if d.IsDir() && !isRoot && !opt.Hidden && (fsutil.IsSkipDir(d.Name(), fsutil.DefaultSkipDirs) || fsutil.IsHidden(p)) {
				return fs.SkipDir
			}
			if isRoot {
				return nil
			}
			if !d.IsDir() && !opt.Hidden && fsutil.IsHidden(p) {
				return nil
			}
			res.Scanned++
			if d.IsDir() && opt.Type == "f" || !d.IsDir() && opt.Type == "d" {
				return nil
			}
			if !d.IsDir() && !d.Type().IsRegular() {
				return nil
			}
			if len(exts) > 0 && (d.IsDir() || !exts[strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))]) {
				return nil
			}
			if !matchName(opt.Pattern, d.Name()) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				if opt.Larger != nil && info.Size() <= *opt.Larger || opt.Smaller != nil && info.Size() >= *opt.Smaller {
					return nil
				}
			}
			age := opt.Now.Sub(info.ModTime())
			if opt.Newer > 0 && age > opt.Newer || opt.Older > 0 && age < opt.Older {
				return nil
			}
			h := Hit{Path: p, Name: d.Name(), IsDir: d.IsDir(), Bytes: info.Size(), Modified: info.ModTime()}
			if d.IsDir() {
				h.Bytes = 0
			}
			if opt.Content != "" {
				if d.IsDir() || info.Size() > maxContentBytes {
					return nil
				}
				select {
				case jobs <- h:
				case <-ctx.Done():
					return errStop
				}
				return nil
			}
			res.Hits = append(res.Hits, h)
			if len(res.Hits) >= opt.Max {
				res.Truncated = true
				return errStop
			}
			return nil
		})
	}

	if opt.Content == "" {
		if err := walk(); err != nil && !errors.Is(err, errStop) {
			return res, err
		}
	} else {
		walkErr := make(chan error, 1)
		go func() {
			err := walk()
			close(jobs)
			walkErr <- err
		}()
		for h := range results {
			res.Hits = append(res.Hits, h)
			if len(res.Hits) >= opt.Max {
				res.Truncated = true
				cancel()
				break
			}
		}
		cancel()
		for range results { // xả để worker thoát
		}
		if err := <-walkErr; err != nil && !errors.Is(err, errStop) {
			return res, err
		}
	}
	if err := parent.Err(); err != nil { // người dùng hủy (Ctrl+C), khác với việc ta tự hủy khi đủ --max
		return res, err
	}
	sort.Slice(res.Hits, func(i, j int) bool { return res.Hits[i].Path < res.Hits[j].Path })
	return res, nil
}

// searchContent tìm chuỗi trong file văn bản (UTF-8/ASCII, hoặc UTF-16 có BOM). Trả dòng khớp đầu tiên.
func searchContent(path, needle string, ignoreCase bool) (int, string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, "", false
	}
	defer f.Close()
	head := make([]byte, 8192)
	n, _ := f.Read(head)
	head = head[:n]
	utf16LE := len(head) >= 2 && head[0] == 0xFF && head[1] == 0xFE
	if !utf16LE && bytes.IndexByte(head, 0) >= 0 {
		return 0, "", false // nhị phân
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, "", false
	}
	norm := func(s string) string {
		if ignoreCase {
			return strings.ToLower(s)
		}
		return s
	}
	want := norm(needle)
	check := func(line string, no int) (int, string, bool) {
		if i := strings.Index(norm(line), want); i >= 0 {
			return no, snippet(line, i, len(want)), true
		}
		return 0, "", false
	}
	if utf16LE {
		raw, err := io.ReadAll(f)
		if err != nil || len(raw) < 2 {
			return 0, "", false
		}
		u := make([]uint16, (len(raw)-2)/2)
		for i := range u {
			u[i] = uint16(raw[2+2*i]) | uint16(raw[3+2*i])<<8
		}
		for i, line := range strings.Split(string(utf16.Decode(u)), "\n") {
			if no, s, ok := check(strings.TrimRight(line, "\r"), i+1); ok {
				return no, s, true
			}
		}
		return 0, "", false
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for no := 1; sc.Scan(); no++ {
		if l, s, ok := check(sc.Text(), no); ok {
			return l, s, true
		}
	}
	return 0, "", false
}

// snippet: tối đa 160 ký tự quanh chỗ khớp.
func snippet(line string, idx, n int) string {
	line = strings.TrimSpace(line)
	r := []rune(line)
	if len(r) <= 160 {
		return string(r)
	}
	pos := len([]rune(line[:min(idx, len(line))]))
	start := max(0, pos-60)
	end := min(len(r), start+160)
	return string(r[start:end])
}
