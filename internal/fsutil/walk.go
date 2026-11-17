package fsutil

import (
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

// DirSize cộng dung lượng mọi file thường dưới root. Lỗi từng mục (không đủ quyền...) được bỏ qua và đếm vào skipped.
func DirSize(root string) (size int64, files int, skipped int) {
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error { // callback luôn trả nil, lỗi từng mục đã đếm vào skipped
		if err != nil {
			skipped++
			return nil
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				skipped++
				return nil
			}
			size += info.Size()
			files++
		}
		return nil
	})
	return
}

// Exists: đường dẫn có tồn tại không.
func Exists(p string) bool { _, err := os.Stat(p); return err == nil }
