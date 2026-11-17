package sysinfo

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type Info struct {
	OS           string   `json:"os"`
	DisplayVer   string   `json:"displayVersion"`
	Build        string   `json:"build"`
	Arch         string   `json:"arch"`
	Hostname     string   `json:"hostname"`
	User         string   `json:"user"`
	IsAdmin      bool     `json:"isAdmin"`
	CPU          string   `json:"cpu"`
	Cores        int      `json:"cores"`
	RAMBytes     uint64   `json:"ramBytes"`
	RAMFreeBytes uint64   `json:"ramFreeBytes"`
	RAMUsedPct   uint32   `json:"ramUsedPct"`
	BootTime     string   `json:"bootTime"`
	Uptime       string   `json:"uptime"`
	UptimeSec    int64    `json:"uptimeSec"`
	Warnings     []string `json:"warnings,omitempty"` // nguồn dữ liệu nào lỗi (mỗi nguồn lỗi độc lập)
}

// ErrUnsupported: chỉ có trên Windows.
var ErrUnsupported = errors.New("chỉ hỗ trợ Windows")

// FormatUptime: 59s → "59 giây", 3h5m → "3 giờ 5 phút", 3d4h12m → "3 ngày 4 giờ 12 phút".
func FormatUptime(d time.Duration) string {
	total := int64(d.Seconds())
	days, hours, mins := total/86400, total%86400/3600, total%3600/60
	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d ngày", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d giờ", hours))
	}
	if mins > 0 {
		parts = append(parts, fmt.Sprintf("%d phút", mins))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d giây", total)
	}
	return strings.Join(parts, " ")
}

// FixWindowsName: ProductName trên Windows 11 vẫn ghi "Windows 10 ..." (lỗi đã biết của Microsoft) → build ≥ 22000 thì sửa.
func FixWindowsName(product string, build int) string {
	if build >= 22000 {
		return strings.Replace(product, "Windows 10", "Windows 11", 1)
	}
	return product
}

// MachineArch đổi PROCESSOR_ARCHITECTURE sang tên Go.
func MachineArch(env string) string {
	switch strings.ToUpper(env) {
	case "AMD64", "X64":
		return "amd64"
	case "ARM64":
		return "arm64"
	case "X86":
		return "386"
	}
	return strings.ToLower(env)
}
