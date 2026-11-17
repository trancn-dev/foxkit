package lock

import (
	"fmt"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"

	"foxkit/internal/winapi"

	"golang.org/x/sys/windows"
)

// Restart Manager không có trong x/sys/windows nên gọi qua DLL. Các hàm Rm* trả mã lỗi trực tiếp (0 = thành công), không dùng GetLastError.
var (
	rstrtmgr                = windows.NewLazySystemDLL("rstrtmgr.dll")
	procRmStartSession      = rstrtmgr.NewProc("RmStartSession")
	procRmRegisterResources = rstrtmgr.NewProc("RmRegisterResources")
	procRmGetList           = rstrtmgr.NewProc("RmGetList")
	procRmEndSession        = rstrtmgr.NewProc("RmEndSession")
)

const (
	errorMoreData   = 234
	cchRmSessionKey = 32
	registerBatch   = 1000
)

// rmUniqueProcess, rmProcessInfo khớp layout C (RM_UNIQUE_PROCESS, RM_PROCESS_INFO) — kiểm kích thước trong test.
type rmUniqueProcess struct {
	ProcessId        uint32
	ProcessStartTime windows.Filetime
}

type rmProcessInfo struct {
	Process          rmUniqueProcess
	AppName          [256]uint16
	ServiceShortName [64]uint16
	ApplicationType  uint32
	AppStatus        uint32
	TSSessionId      uint32
	Restartable      int32
}

func rmErr(name string, r1 uintptr) error {
	return fmt.Errorf("%s lỗi: %w", name, syscall.Errno(r1))
}

// Find hỏi Restart Manager process nào đang giữ các file/thư mục.
func Find(paths []string, maxFiles int) (Result, error) {
	res := Result{Paths: paths, Holders: []Holder{}}
	files, truncated, err := Expand(paths, maxFiles)
	if err != nil {
		return res, err
	}
	res.Checked, res.Truncated = len(files), truncated
	if len(files) == 0 {
		return res, nil
	}

	var h uint32
	var key [cchRmSessionKey + 1]uint16
	if r1, _, _ := procRmStartSession.Call(uintptr(unsafe.Pointer(&h)), 0, uintptr(unsafe.Pointer(&key[0]))); r1 != 0 {
		return res, rmErr("RmStartSession", r1)
	}
	defer procRmEndSession.Call(uintptr(h))

	for _, batch := range Batches(files, registerBatch) {
		ptrs := make([]*uint16, len(batch))
		for i, f := range batch {
			if ptrs[i], err = windows.UTF16PtrFromString(f); err != nil {
				return res, err
			}
		}
		r1, _, _ := procRmRegisterResources.Call(uintptr(h), uintptr(len(ptrs)), uintptr(unsafe.Pointer(&ptrs[0])), 0, 0, 0, 0)
		runtime.KeepAlive(ptrs) // giữ mảng con trỏ chuỗi sống tới sau lời gọi để GC không thu hồi
		if r1 != 0 {
			return res, rmErr("RmRegisterResources", r1)
		}
	}

	// gọi 2 lần: lần 1 hỏi cần bao nhiêu phần tử, lần 2 lấy dữ liệu; process mới xuất hiện giữa hai lần thì lặp tối đa 3 lần
	var infos []rmProcessInfo
	var needed, count, reasons uint32
	for attempt := 0; attempt < 3; attempt++ {
		count = uint32(len(infos))
		var first *rmProcessInfo
		if len(infos) > 0 {
			first = &infos[0]
		}
		r1, _, _ := procRmGetList.Call(uintptr(h), uintptr(unsafe.Pointer(&needed)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(first)), uintptr(unsafe.Pointer(&reasons)))
		if r1 == 0 {
			infos = infos[:count]
			break
		}
		if r1 != errorMoreData {
			return res, rmErr("RmGetList", r1)
		}
		infos = make([]rmProcessInfo, needed)
		if attempt == 2 {
			return res, fmt.Errorf("RmGetList: danh sách process đổi liên tục, thử lại")
		}
	}

	seen := map[uint32]bool{}
	for _, in := range infos {
		pid := in.Process.ProcessId
		if seen[pid] {
			continue
		}
		seen[pid] = true
		ho := Holder{PID: pid, App: windows.UTF16ToString(in.AppName[:]), Service: windows.UTF16ToString(in.ServiceShortName[:]), Type: TypeName(in.ApplicationType)}
		if p, err := winapi.ProcessPath(int(pid)); err == nil {
			ho.Path, ho.Name = p, filepath.Base(p)
		} else {
			ho.Name = ho.App // thiếu quyền mở process thì dùng tên hiển thị
		}
		res.Holders = append(res.Holders, ho)
	}
	return res, nil
}
