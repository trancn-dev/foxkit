package syncdir

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Action struct {
	Op     string `json:"op"` // copy | update | delete | mkdir
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	Reason string `json:"reason,omitempty"`
}

type Failure struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type Plan struct {
	Applied   bool      `json:"applied"`
	Src       string    `json:"src"`
	Dst       string    `json:"dst"`
	Actions   []Action  `json:"actions"`
	Same      int       `json:"same"`
	CopyBytes int64     `json:"copyBytes"`
	Failed    []Failure `json:"failed"`
	// OnlyInDst: file/thư mục chỉ có ở đích (không bị đụng nếu không có --delete).
	OnlyInDst int `json:"onlyInDst"`
}

type Options struct {
	Delete            bool
	Exclude           []string
	NoDefaultExcludes bool
	Compare           string // size | hash
	Now               func() time.Time
}

var (
	ErrOverlap   = errors.New("nguồn và đích trùng hoặc lồng nhau")
	ErrDstUnsafe = errors.New("đích không an toàn")
	ErrNotDir    = errors.New("nguồn không phải thư mục")
)

// DiskFullError: CopyBytes lớn hơn dung lượng trống của ổ đích.
type DiskFullError struct{ Need, Have int64 }

func (e *DiskFullError) Error() string {
	return fmt.Sprintf("đích không đủ chỗ: cần %d byte, còn %d", e.Need, e.Have)
}

const (
	trashDir = ".foxkit-trash"
	tmpExt   = ".foxkit-tmp"
	mtimeTol = 2 * time.Second // FAT/exFAT lưu mtime thô nên dung sai 2 giây
)

var defaultExcludes = []string{"Thumbs.db", "desktop.ini", "*.tmp", "$RECYCLE.BIN", "System Volume Information"}

type node struct {
	rel   string // đường dẫn tương đối gốc, dấu /
	isDir bool
	size  int64
	mtime time.Time
}

func lower(s string) string { return strings.ToLower(s) }

func within(parent, child string) bool {
	p, c := lower(filepath.Clean(parent)), lower(filepath.Clean(child))
	return p == c || strings.HasPrefix(c, p+string(filepath.Separator))
}

var driveRoot = regexp.MustCompile(`^[A-Za-z]:\\?$`)

// UnsafeDst: gốc ổ đĩa, C:\Windows, C:\Program Files, thư mục Users gốc.
func UnsafeDst(abs string) bool {
	c := filepath.Clean(abs)
	if driveRoot.MatchString(c) || c == filepath.VolumeName(c)+`\` {
		return true
	}
	for _, bad := range []string{`C:\Windows`, `C:\Program Files`, `C:\Program Files (x86)`, `C:\Users`, `C:\ProgramData`} {
		if strings.EqualFold(c, bad) {
			return true
		}
	}
	return false
}

func (o Options) excluded(name, rel string) bool {
	pats := append([]string{}, o.Exclude...)
	if !o.NoDefaultExcludes {
		pats = append(pats, defaultExcludes...)
	}
	pats = append(pats, "*"+tmpExt)
	ln, lr := lower(name), lower(rel)
	for _, p := range pats {
		lp := lower(p)
		if ok, _ := path.Match(lp, ln); ok {
			return true
		}
		if ok, _ := path.Match(lp, lr); ok {
			return true
		}
	}
	return false
}

func scan(root string, opt Options, isDst bool) (map[string]node, error) {
	out := map[string]node{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." {
			return nil
		}
		slash := filepath.ToSlash(rel)
		if isDst && lower(strings.SplitN(slash, "/", 2)[0]) == trashDir {
			return fs.SkipDir // thùng rác foxkit ở đích tự bị loại khỏi so sánh
		}
		if opt.excluded(d.Name(), slash) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		out[lower(slash)] = node{rel: slash, isDir: d.IsDir(), size: info.Size(), mtime: info.ModTime()}
		return nil
	})
	return out, err
}

func fileHash(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func absDir(p string) (string, error) {
	a, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return filepath.Clean(a), nil
}

// Build so sánh hai cây và lập kế hoạch (không ghi gì). Mặc định không xóa ở đích.
func Build(ctx context.Context, src, dst string, opt Options) (Plan, error) {
	p := Plan{Actions: []Action{}, Failed: []Failure{}}
	as, err := absDir(src)
	if err != nil {
		return p, err
	}
	ad, err := absDir(dst)
	if err != nil {
		return p, err
	}
	p.Src, p.Dst = as, ad
	if st, err := os.Stat(as); err != nil {
		return p, err
	} else if !st.IsDir() {
		return p, ErrNotDir
	}
	if within(as, ad) || within(ad, as) {
		return p, ErrOverlap
	}
	if UnsafeDst(ad) {
		return p, ErrDstUnsafe
	}
	if opt.Compare == "" {
		opt.Compare = "size"
	}
	A, err := scan(as, opt, false)
	if err != nil {
		return p, err
	}
	B := map[string]node{}
	if _, err := os.Stat(ad); err == nil {
		if B, err = scan(ad, opt, true); err != nil {
			return p, err
		}
	}
	keys := make([]string, 0, len(A))
	for k := range A {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var mkdirs, work []Action
	if _, err := os.Stat(ad); err != nil {
		mkdirs = append(mkdirs, Action{Op: "mkdir", Path: ""}) // đích chưa có
	}
	for _, k := range keys {
		if err := ctx.Err(); err != nil {
			return p, err
		}
		a := A[k]
		b, inB := B[k]
		switch {
		case a.isDir && !inB:
			mkdirs = append(mkdirs, Action{Op: "mkdir", Path: a.rel})
		case a.isDir && inB && !b.isDir, !a.isDir && inB && b.isDir:
			p.Failed = append(p.Failed, Failure{Path: a.rel, Reason: "ở nguồn và đích khác kiểu (file/thư mục)"})
		case a.isDir:
			// thư mục đã có ở cả hai
		case !inB:
			work = append(work, Action{Op: "copy", Path: a.rel, Bytes: a.size, Reason: "mới"})
			p.CopyBytes += a.size
		default:
			reason := ""
			switch {
			case a.size != b.size:
				reason = "size khác"
			case opt.Compare == "hash":
				ha, e1 := fileHash(filepath.Join(as, filepath.FromSlash(a.rel)))
				hb, e2 := fileHash(filepath.Join(ad, filepath.FromSlash(b.rel)))
				if e1 != nil || e2 != nil {
					p.Failed = append(p.Failed, Failure{Path: a.rel, Reason: "không đọc được để so hash"})
					continue
				}
				if ha != hb {
					reason = "hash khác"
				}
			default:
				if d := a.mtime.Sub(b.mtime); d > mtimeTol || d < -mtimeTol {
					reason = "mtime khác"
				}
			}
			if reason == "" {
				p.Same++
				continue
			}
			work = append(work, Action{Op: "update", Path: a.rel, Bytes: a.size, Reason: reason})
			p.CopyBytes += a.size
		}
	}
	// file/thư mục chỉ có ở đích: chỉ xóa khi --delete (vào thùng rác); thư mục thừa thì chuyển cả cây một lần
	var extra []string
	for k, b := range B {
		if _, inA := A[k]; !inA {
			extra = append(extra, b.rel)
		}
	}
	sort.Strings(extra)
	p.OnlyInDst = len(extra)
	if opt.Delete {
		var dels []Action
		for _, rel := range extra {
			if len(dels) > 0 && strings.HasPrefix(lower(rel), lower(dels[len(dels)-1].Path)+"/") {
				continue // đã nằm trong thư mục thừa sắp chuyển
			}
			dels = append(dels, Action{Op: "delete", Path: rel, Bytes: B[lower(rel)].size, Reason: "chỉ có ở đích"})
		}
		work = append(work, dels...)
	}
	sort.SliceStable(mkdirs, func(i, j int) bool { return strings.Count(mkdirs[i].Path, "/") < strings.Count(mkdirs[j].Path, "/") })
	p.Actions = append(append(p.Actions, mkdirs...), work...)

	if p.CopyBytes > 0 {
		if free, ok := freeBytes(nearestExisting(ad)); ok && uint64(p.CopyBytes) > free {
			return p, &DiskFullError{Need: p.CopyBytes, Have: int64(free)}
		}
	}
	return p, nil
}

// nearestExisting: tổ tiên gần nhất đã tồn tại (đích có thể chưa được tạo) để hỏi dung lượng trống.
func nearestExisting(p string) string {
	for {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		parent := filepath.Dir(p)
		if parent == p {
			return p
		}
		p = parent
	}
}
