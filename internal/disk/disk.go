package disk

import (
	"errors"
	"strings"
)

type Drive struct {
	Root       string  `json:"root"`
	Type       string  `json:"type"`
	Label      string  `json:"label"`
	FS         string  `json:"fs"`
	TotalBytes uint64  `json:"totalBytes"`
	FreeBytes  uint64  `json:"freeBytes"`
	UsedPct    float64 `json:"usedPct"`
	Warn       bool    `json:"warn"`
	Error      string  `json:"error,omitempty"`
}

// ErrUnsupported: chỉ có trên Windows.
var ErrUnsupported = errors.New("chỉ hỗ trợ Windows")

// SplitMultiSZ tách chuỗi nhiều phần kết thúc bằng NUL kép: "C:\\x00D:\\x00\x00" → ["C:\\","D:\\"].
func SplitMultiSZ(s string) []string {
	var out []string
	for _, p := range strings.Split(s, "\x00") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Fill tính UsedPct (làm tròn 1 chữ số, dựa trên dung lượng user được dùng) và Warn.
func (d *Drive) Fill(warnPct float64) {
	if d.TotalBytes == 0 {
		return
	}
	used := d.TotalBytes - d.FreeBytes
	d.UsedPct = float64(int(float64(used)/float64(d.TotalBytes)*1000+0.5)) / 10
	d.Warn = d.UsedPct >= warnPct
}

// TypeName đổi mã DRIVE_* của Windows sang tên.
func TypeName(t uint32) string {
	switch t {
	case 2:
		return "removable"
	case 3:
		return "fixed"
	case 4:
		return "network"
	case 5:
		return "cdrom"
	case 6:
		return "ram"
	}
	return "unknown"
}
