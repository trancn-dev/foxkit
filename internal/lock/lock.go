package lock

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type Holder struct {
	PID     uint32 `json:"pid"`
	Name    string `json:"name"`
	App     string `json:"app"`
	Service string `json:"service,omitempty"`
	Type    string `json:"type"`
	Path    string `json:"path,omitempty"`
}

type Result struct {
	Paths     []string `json:"paths"`
	Checked   int      `json:"checked"`
	Truncated bool     `json:"truncated"`
	Holders   []Holder `json:"holders"`
	Killed    []int    `json:"killed,omitempty"`
}

// ErrUnsupported: chỉ có trên Windows.
var ErrUnsupported = errors.New("chỉ hỗ trợ Windows")

// TypeName đổi RM_APP_TYPE sang tên.
func TypeName(t uint32) string {
	switch t {
	case 1:
		return "mainWindow"
	case 2:
		return "otherWindow"
	case 3:
		return "service"
	case 4:
		return "explorer"
	case 5:
		return "console"
	case 1000:
		return "critical"
	}
	return "unknown"
}

// Expand mở rộng đầu vào thành danh sách file tuyệt đối: file giữ nguyên, thư mục duyệt (không đi vào junction/symlink)
// tối đa maxFiles file; quá thì truncated=true.
func Expand(paths []string, maxFiles int) (files []string, truncated bool, err error) {
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, false, err
		}
		st, err := os.Stat(abs)
		if err != nil {
			return nil, false, err
		}
		if !st.IsDir() {
			files = append(files, abs)
			continue
		}
		err = filepath.WalkDir(abs, func(q string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil // mục không đọc được (quyền) thì bỏ qua
			}
			if d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0 {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if len(files) >= maxFiles {
				truncated = true
				return fs.SkipAll
			}
			files = append(files, q)
			return nil
		})
		if err != nil {
			return nil, false, err
		}
	}
	return files, truncated, nil
}

// Batches chia danh sách thành lô tối đa n phần tử (đăng ký quá nhiều file một lần có thể lỗi).
func Batches(files []string, n int) [][]string {
	var out [][]string
	for len(files) > 0 {
		k := min(n, len(files))
		out = append(out, files[:k])
		files = files[k:]
	}
	return out
}
