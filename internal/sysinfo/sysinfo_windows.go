package sysinfo

import (
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"foxkit/internal/winapi"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// memoryStatusEx khớp layout MEMORYSTATUSEX của C (64 byte).
type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetTickCount64       = kernel32.NewProc("GetTickCount64")
)

func memory() (memoryStatusEx, error) {
	var ms memoryStatusEx
	ms.Length = uint32(unsafe.Sizeof(ms)) // API đòi biết kích thước struct trước khi điền
	r1, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&ms)))
	if r1 == 0 {
		return ms, err
	}
	return ms, nil
}

// Get gom thông tin máy; mỗi nguồn lỗi độc lập → field đó rỗng và thêm vào Warnings.
func Get() (Info, error) {
	in := Info{Cores: runtime.NumCPU()}
	warn := func(what string, err error) { in.Warnings = append(in.Warnings, fmt.Sprintf("%s: %v", what, err)) }
	failed := 0

	build := 0
	if v := windows.RtlGetVersion(); v != nil {
		build = int(v.BuildNumber)
		in.Build = fmt.Sprint(v.BuildNumber)
	}
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE); err == nil {
		product, _, _ := k.GetStringValue("ProductName")
		in.OS = FixWindowsName(strings.TrimSpace(product), build)
		in.DisplayVer, _, _ = k.GetStringValue("DisplayVersion")
		if ubr, _, err := k.GetIntegerValue("UBR"); err == nil {
			in.Build = fmt.Sprintf("%d.%d", build, ubr)
		}
		k.Close()
	} else {
		warn("Windows (registry)", err)
		failed++
	}
	arch := os.Getenv("PROCESSOR_ARCHITEW6432")
	if arch == "" {
		arch = os.Getenv("PROCESSOR_ARCHITECTURE")
	}
	in.Arch = MachineArch(arch)
	if in.Arch == "" {
		in.Arch = runtime.GOARCH
	}
	if h, err := os.Hostname(); err == nil {
		in.Hostname = h
	} else {
		warn("hostname", err)
	}
	if u, err := user.Current(); err == nil {
		in.User = u.Username
	} else {
		warn("user", err)
	}
	in.IsAdmin = winapi.IsAdmin()
	if k, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DESCRIPTION\System\CentralProcessor\0`, registry.QUERY_VALUE); err == nil {
		name, _, _ := k.GetStringValue("ProcessorNameString")
		in.CPU = strings.Join(strings.Fields(name), " ") // gộp khoảng trắng thừa
		k.Close()
	} else {
		warn("CPU (registry)", err)
		failed++
	}
	if ms, err := memory(); err == nil {
		in.RAMBytes, in.RAMFreeBytes, in.RAMUsedPct = ms.TotalPhys, ms.AvailPhys, ms.MemoryLoad
	} else {
		warn("RAM", err)
		failed++
	}
	if r1, _, _ := procGetTickCount64.Call(); r1 != 0 {
		up := time.Duration(r1) * time.Millisecond
		in.UptimeSec = int64(up.Seconds())
		in.Uptime = FormatUptime(up)
		in.BootTime = time.Now().Add(-up).Format("2006-01-02 15:04")
	} else {
		warn("uptime", fmt.Errorf("GetTickCount64 trả 0"))
		failed++
	}
	if failed >= 4 {
		return in, fmt.Errorf("mọi nguồn dữ liệu đều lỗi")
	}
	return in, nil
}
