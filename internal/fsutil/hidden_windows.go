package fsutil

import (
	"strings"
	"syscall"
)

// IsHidden: tên bắt đầu bằng "." hoặc có thuộc tính Hidden của NTFS.
func IsHidden(path string) bool {
	base := path[strings.LastIndexAny(path, `\/`)+1:]
	if strings.HasPrefix(base, ".") {
		return true
	}
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}
	attr, err := syscall.GetFileAttributes(p)
	return err == nil && attr&syscall.FILE_ATTRIBUTE_HIDDEN != 0
}
