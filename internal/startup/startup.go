package startup

import (
	"errors"
	"sort"
	"strings"
)

type Item struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	Source  string `json:"source"`
	Scope   string `json:"scope"` // user | machine
	Enabled bool   `json:"enabled"`
	Exists  bool   `json:"exists"`
}

var (
	// ErrUnsupported: chỉ có trên Windows.
	ErrUnsupported = errors.New("chỉ hỗ trợ Windows")
	// ErrAllFailed: mọi nguồn đều lỗi.
	ErrAllFailed = errors.New("mọi nguồn khởi động đều đọc lỗi")
)

// ExePath tách đường dẫn exe khỏi command: có nháy kép thì lấy phần trong nháy, không thì tới ".exe" đầu tiên (hoặc token đầu).
func ExePath(command string) string {
	c := strings.TrimSpace(command)
	if strings.HasPrefix(c, `"`) {
		if end := strings.Index(c[1:], `"`); end >= 0 {
			return c[1 : 1+end]
		}
		return strings.Trim(c, `"`)
	}
	if i := strings.Index(strings.ToLower(c), ".exe"); i >= 0 {
		return c[:i+4]
	}
	if f := strings.Fields(c); len(f) > 0 {
		return f[0]
	}
	return c
}

// Approved đọc giá trị nhị phân của StartupApproved: byte đầu chẵn (0x02, 0x06) = bật, lẻ (0x03) = tắt; không có giá trị = bật.
func Approved(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	return b[0]&1 == 0
}

// Sort: bật trước, rồi theo tên.
func Sort(items []Item) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Enabled != items[j].Enabled {
			return items[i].Enabled
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
}
