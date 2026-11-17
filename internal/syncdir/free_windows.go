package syncdir

import "golang.org/x/sys/windows"

// freeBytes: dung lượng trống của ổ chứa p (tính theo quota của user hiện tại).
func freeBytes(p string) (uint64, bool) {
	ptr, err := windows.UTF16PtrFromString(p)
	if err != nil {
		return 0, false
	}
	var avail, total, free uint64
	if err := windows.GetDiskFreeSpaceEx(ptr, &avail, &total, &free); err != nil {
		return 0, false
	}
	return avail, true
}
