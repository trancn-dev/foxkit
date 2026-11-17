//go:build !windows

package fsutil

import "strings"

// IsHidden: tên bắt đầu bằng ".".
func IsHidden(path string) bool {
	base := path[strings.LastIndexAny(path, `\/`)+1:]
	return strings.HasPrefix(base, ".")
}
