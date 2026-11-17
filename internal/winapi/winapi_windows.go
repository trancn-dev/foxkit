package winapi

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// IsAdmin: tiến trình hiện tại có đang chạy với quyền Administrator (elevated) không.
func IsAdmin() bool { return windows.GetCurrentProcessToken().IsElevated() }

// ProcessPath trả đường dẫn exe của process; lỗi quyền (process hệ thống/của user khác) trả error.
func ProcessPath(pid int) (string, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(h)
	var buf [windows.MAX_LONG_PATH]uint16
	size := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &size); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buf[:size]), nil
}

// Processes liệt kê process bằng CreateToolhelp32Snapshot (tên Unicode đúng, có PPID).
func Processes() ([]Proc, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe)) // API đòi biết kích thước struct trước khi điền
	out := []Proc{}
	for err = windows.Process32First(snap, &pe); err == nil; err = windows.Process32Next(snap, &pe) {
		out = append(out, Proc{PID: int(pe.ProcessID), PPID: int(pe.ParentProcessID), Name: windows.UTF16ToString(pe.ExeFile[:])})
	}
	if err != windows.ERROR_NO_MORE_FILES {
		return out, err
	}
	return out, nil
}
