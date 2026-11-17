package diff

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"foxkit/internal/fsutil"
)

type FileDiff struct {
	Path    string `json:"path"`
	Status  string `json:"status"` // same | changed | added | removed | binary
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Hunks   []Hunk `json:"hunks,omitempty"`
}

// Result: với thư mục Added/Removed/Changed là số file; với một file là số dòng thêm/xóa và Changed = 0/1.
type Result struct {
	A         string     `json:"a"`
	B         string     `json:"b"`
	Identical bool       `json:"identical"`
	Files     []FileDiff `json:"files"`
	Added     int        `json:"added"`
	Removed   int        `json:"removed"`
	Changed   int        `json:"changed"`
}

const hashOnlyBytes = 10 << 20

func hashFile(p string) ([32]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return [32]byte{}, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return [32]byte{}, err
	}
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out, nil
}

func isBinary(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 8192)
	n, _ := f.Read(buf)
	return bytes.IndexByte(buf[:n], 0) >= 0
}

func splitLines(b []byte) []string {
	s := string(b)
	if s == "" {
		return nil
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

// Files so sánh hai file: nhị phân/quá lớn chỉ so hash; còn lại diff theo dòng.
func Files(aPath, bPath string, opt Options) (FileDiff, error) {
	fd := FileDiff{Path: filepath.Base(bPath), Status: "same"}
	sa, err := os.Stat(aPath)
	if err != nil {
		return fd, err
	}
	sb, err := os.Stat(bPath)
	if err != nil {
		return fd, err
	}
	bin := isBinary(aPath) || isBinary(bPath)
	if bin || sa.Size() > hashOnlyBytes || sb.Size() > hashOnlyBytes {
		ha, err := hashFile(aPath)
		if err != nil {
			return fd, err
		}
		hb, err := hashFile(bPath)
		if err != nil {
			return fd, err
		}
		if ha != hb {
			fd.Status = "changed" // quá lớn: chỉ báo khác, không in nội dung
			if bin {
				fd.Status = "binary"
			}
		}
		return fd, nil
	}
	da, err := os.ReadFile(aPath)
	if err != nil {
		return fd, err
	}
	db, err := os.ReadFile(bPath)
	if err != nil {
		return fd, err
	}
	ops := Lines(splitLines(da), splitLines(db), opt)
	for _, l := range ops {
		switch l.Op {
		case Add:
			fd.Added++
		case Del:
			fd.Removed++
		}
	}
	if fd.Added+fd.Removed > 0 {
		fd.Status = "changed"
		fd.Hunks = ToHunks(ops, opt.Context)
	}
	return fd, nil
}

func listFiles(root string, opt Options) (map[string]string, error) {
	out := map[string]string{} // khóa chữ thường (Windows không phân biệt hoa thường) → đường dẫn tương đối gốc
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if rel == "." {
			return nil
		}
		for _, g := range opt.Exclude {
			if ok, _ := path.Match(g, d.Name()); ok {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			if fsutil.IsSkipDir(d.Name(), fsutil.DefaultSkipDirs) || fsutil.Skippable(d) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type().IsRegular() {
			rel = filepath.ToSlash(rel)
			out[strings.ToLower(rel)] = rel
		}
		return nil
	})
	return out, err
}

// Dirs so sánh hai cây thư mục theo đường dẫn tương đối.
func Dirs(a, b string, opt Options) (Result, error) {
	res := Result{A: a, B: b, Files: []FileDiff{}}
	la, err := listFiles(a, opt)
	if err != nil {
		return res, err
	}
	lb, err := listFiles(b, opt)
	if err != nil {
		return res, err
	}
	keys := map[string]bool{}
	for k := range la {
		keys[k] = true
	}
	for k := range lb {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)
	for _, k := range sorted {
		ra, inA := la[k]
		rb, inB := lb[k]
		switch {
		case inA && !inB:
			res.Files = append(res.Files, FileDiff{Path: ra, Status: "removed"})
			res.Removed++
		case !inA && inB:
			res.Files = append(res.Files, FileDiff{Path: rb, Status: "added"})
			res.Added++
		default:
			pa, pb := filepath.Join(a, filepath.FromSlash(ra)), filepath.Join(b, filepath.FromSlash(rb))
			fd, err := compare(pa, pb, opt)
			if err != nil {
				return res, fmt.Errorf("%s: %w", rb, err)
			}
			fd.Path = rb
			if fd.Status != "same" {
				res.Changed++
				res.Files = append(res.Files, fd)
			}
		}
	}
	res.Identical = len(res.Files) == 0
	return res, nil
}

// compare: --names-only chỉ cần biết khác hay không (size rồi hash); còn lại diff dòng.
func compare(pa, pb string, opt Options) (FileDiff, error) {
	if !opt.NamesOnly {
		return Files(pa, pb, opt)
	}
	fd := FileDiff{Status: "same"}
	sa, err := os.Stat(pa)
	if err != nil {
		return fd, err
	}
	sb, err := os.Stat(pb)
	if err != nil {
		return fd, err
	}
	if sa.Size() != sb.Size() {
		fd.Status = "changed"
		return fd, nil
	}
	ha, err := hashFile(pa)
	if err != nil {
		return fd, err
	}
	hb, err := hashFile(pb)
	if err != nil {
		return fd, err
	}
	if ha != hb {
		fd.Status = "changed"
	}
	return fd, nil
}
