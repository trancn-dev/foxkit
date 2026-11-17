package fsutil

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// IsSkipDir: tên thư mục có nằm trong danh sách bỏ qua không (không phân biệt hoa thường).
func IsSkipDir(name string, skip []string) bool {
	for _, s := range skip {
		if strings.EqualFold(name, s) {
			return true
		}
	}
	return false
}

// Exists: đường dẫn có tồn tại không.
func Exists(p string) bool { _, err := os.Stat(p); return err == nil }

// IsDir: p có phải thư mục không.
func IsDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// Skippable: symlink/junction/irregular — không đi theo, không tính dung lượng.
func Skippable(d fs.DirEntry) bool {
	return d.Type()&(fs.ModeSymlink|fs.ModeIrregular) != 0
}

type DirStat struct {
	Bytes  int64
	Files  int
	Dirs   int
	Denied int
}

// DirSize cộng dung lượng mọi file thường dưới root. Lỗi quyền từng mục → Denied++ và đi tiếp.
// Chỉ trả error khi ctx bị hủy.
func DirSize(ctx context.Context, root string) (DirStat, error) {
	var st DirStat
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			st.Denied++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		switch {
		case Skippable(d):
			if d.IsDir() {
				return fs.SkipDir
			}
		case d.IsDir():
			if p != root {
				st.Dirs++
			}
		case d.Type().IsRegular():
			info, err := d.Info()
			if err != nil {
				st.Denied++
				return nil
			}
			st.Bytes += info.Size()
			st.Files++
		}
		return nil
	})
	if ctx.Err() != nil {
		return st, ctx.Err()
	}
	return st, err
}
