package zipx

import (
	"archive/zip"
	"compress/flate"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PackOptions struct {
	Exclude           []string // glob tên hoặc đường dẫn tương đối; thêm vào danh sách mặc định
	NoDefaultExcludes bool
	Level             int // 0 = chỉ lưu
	Force             bool
}

type PackResult struct {
	Out      string   `json:"out"`
	Files    int      `json:"files"`
	Bytes    int64    `json:"bytes"`
	ZipBytes int64    `json:"zipBytes"`
	Excluded []string `json:"excluded"`
	Skipped  []string `json:"skipped"`
}

type Entry struct {
	Name       string    `json:"name"`
	Bytes      int64     `json:"bytes"`
	Compressed int64     `json:"compressed"`
	Modified   time.Time `json:"modified"`
	IsDir      bool      `json:"isDir"`
}

var (
	ErrExists     = errors.New("file đã tồn tại")
	ErrNotFound   = errors.New("không tìm thấy nguồn")
	ErrInvalid    = errors.New("file zip hỏng hoặc không phải zip")
	ErrUnsafePath = errors.New("zip chứa đường dẫn nguy hiểm")
	ErrTooLarge   = errors.New("dung lượng giải nén quá lớn (nghi zip bomb)")
)

// DefaultExcludes: tự bỏ khi nén dự án (kể cả .env để khỏi gửi nhầm bí mật).
var DefaultExcludes = []string{".git", "node_modules", "vendor", ".idea", ".vscode", "*.log", ".env"}

// alreadyCompressed: định dạng đã nén sẵn, dùng Store cho nhanh.
var alreadyCompressed = map[string]bool{".zip": true, ".png": true, ".jpg": true, ".jpeg": true, ".mp4": true, ".gz": true, ".7z": true, ".rar": true, ".webp": true, ".mp3": true}

// excluder khớp theo tên hoặc đường dẫn tương đối (dấu /), không phân biệt hoa thường.
type excluder struct {
	patterns []string
	hits     map[string]int
}

func (e *excluder) match(name, rel string) (string, bool) {
	ln, lr := strings.ToLower(name), strings.ToLower(rel)
	for _, p := range e.patterns {
		lp := strings.ToLower(p)
		if ok, _ := path.Match(lp, ln); ok {
			return p, true
		}
		if ok, _ := path.Match(lp, lr); ok {
			return p, true
		}
	}
	return "", false
}

func (e *excluder) labels() []string {
	out := make([]string, 0, len(e.hits))
	for p, n := range e.hits {
		if n > 1 || strings.ContainsAny(p, "*?") {
			out = append(out, fmt.Sprintf("%s ×%d", p, n))
		} else {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// DefaultOut: <tên nguồn>.zip cạnh nguồn.
func DefaultOut(src string) string {
	abs, err := filepath.Abs(src)
	if err != nil {
		abs = src
	}
	return abs + ".zip"
}

// Pack nén srcs vào out: ghi `<out>.tmp` rồi rename (lỗi giữa chừng không để lại zip hỏng). Đường dẫn trong zip dùng "/", tương đối so với cha của nguồn.
func Pack(ctx context.Context, srcs []string, out string, opt PackOptions) (PackResult, error) {
	res := PackResult{Out: out, Excluded: []string{}, Skipped: []string{}}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return res, err
	}
	if _, err := os.Stat(absOut); err == nil && !opt.Force {
		return res, fmt.Errorf("%w: %s", ErrExists, out)
	}
	ex := &excluder{hits: map[string]int{}}
	if !opt.NoDefaultExcludes {
		ex.patterns = append(ex.patterns, DefaultExcludes...)
	}
	ex.patterns = append(ex.patterns, opt.Exclude...)
	for _, s := range srcs {
		if _, err := os.Stat(s); err != nil {
			return res, fmt.Errorf("%w: %s", ErrNotFound, s)
		}
	}
	tmp := absOut + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return res, err
	}
	ok := false
	defer func() {
		if !ok {
			f.Close()
			_ = os.Remove(tmp) // không để lại zip dở dang
		}
	}()
	zw := zip.NewWriter(f)
	level := opt.Level
	zw.RegisterCompressor(zip.Deflate, func(w io.Writer) (io.WriteCloser, error) { return flate.NewWriter(w, level) })

	for _, src := range srcs {
		absSrc, err := filepath.Abs(src)
		if err != nil {
			return res, err
		}
		parent := filepath.Dir(absSrc)
		err = filepath.WalkDir(absSrc, func(p string, d fs.DirEntry, werr error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			rel, _ := filepath.Rel(parent, p)
			if werr != nil {
				res.Skipped = append(res.Skipped, filepath.ToSlash(rel))
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if strings.EqualFold(p, absOut) || strings.EqualFold(p, tmp) {
				return nil // không nén chính file zip đang ghi
			}
			if p != absSrc { // nguồn do người dùng chỉ định luôn được nén, kể cả tên trùng mẫu loại
				inner, _ := filepath.Rel(absSrc, p) // so mẫu theo đường dẫn bên trong nguồn (storage/framework/cache), không kèm tên thư mục nguồn
				if label, hit := ex.match(d.Name(), filepath.ToSlash(inner)); hit {
					ex.hits[label]++
					if d.IsDir() {
						return fs.SkipDir
					}
					return nil
				}
			}
			if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				res.Skipped = append(res.Skipped, filepath.ToSlash(rel))
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			info, err := d.Info()
			if err != nil {
				res.Skipped = append(res.Skipped, filepath.ToSlash(rel))
				return nil
			}
			name := filepath.ToSlash(rel)
			if d.IsDir() {
				h, err := zip.FileInfoHeader(info)
				if err != nil {
					return err
				}
				h.Name = name + "/"
				h.Method = zip.Store
				_, err = zw.CreateHeader(h)
				return err
			}
			src, err := os.Open(p)
			if err != nil {
				res.Skipped = append(res.Skipped, name) // đang bị khóa: bỏ qua, nén tiếp
				return nil
			}
			defer src.Close()
			h, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			h.Name = name
			h.Method = zip.Deflate
			if level == 0 || alreadyCompressed[strings.ToLower(filepath.Ext(p))] {
				h.Method = zip.Store
			}
			w, err := zw.CreateHeader(h)
			if err != nil {
				return err
			}
			n, err := io.Copy(w, src)
			if err != nil {
				return err
			}
			res.Files++
			res.Bytes += n
			return nil
		})
		if err != nil {
			return res, err
		}
	}
	if err := zw.Close(); err != nil {
		return res, err
	}
	if err := f.Close(); err != nil {
		return res, err
	}
	if opt.Force {
		_ = os.Remove(absOut) // Rename không ghi đè trên Windows
	}
	if err := os.Rename(tmp, absOut); err != nil {
		return res, err
	}
	ok = true
	if st, err := os.Stat(absOut); err == nil {
		res.ZipBytes = st.Size()
	}
	res.Excluded = append(res.Excluded, ex.labels()...)
	return res, nil
}

// List liệt kê mục trong zip.
func List(zipPath string) ([]Entry, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, zipPath)
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	defer r.Close()
	out := make([]Entry, 0, len(r.File))
	for _, f := range r.File {
		out = append(out, Entry{Name: f.Name, Bytes: int64(f.UncompressedSize64), Compressed: int64(f.CompressedSize64), Modified: f.Modified, IsDir: f.FileInfo().IsDir()})
	}
	return out, nil
}
