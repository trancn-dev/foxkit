package fsutil

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DefaultSkipDirs: thư mục thường bỏ qua khi quét dự án.
var DefaultSkipDirs = []string{".git", "node_modules", "vendor", ".idea", ".vscode", "storage", "dist", "build", ".next", ".nuxt"}

var sizeUnits = []struct {
	suffix string
	mult   int64
}{{"TB", 1 << 40}, {"GB", 1 << 30}, {"MB", 1 << 20}, {"KB", 1 << 10}, {"B", 1}}

// ParseSize: "1KB","10MB","2GB","500" (byte), không phân biệt hoa thường, cơ số 1024.
func ParseSize(s string) (int64, error) {
	t := strings.ToUpper(strings.TrimSpace(s))
	mult := int64(1)
	for _, u := range sizeUnits {
		if strings.HasSuffix(t, u.suffix) {
			t, mult = strings.TrimSpace(strings.TrimSuffix(t, u.suffix)), u.mult
			break
		}
	}
	if f, err := strconv.ParseFloat(t, 64); err == nil && f >= 0 && !strings.ContainsAny(t, "EIN+") {
		return int64(f * float64(mult)), nil
	}
	return 0, fmt.Errorf("kích thước không hợp lệ: %q", s)
}

// ParseAge: "90m","12h","30d","2w","1y" (y = 365 ngày; m = phút).
func ParseAge(s string) (time.Duration, error) {
	t := strings.TrimSpace(s)
	if len(t) < 2 {
		return 0, fmt.Errorf("thời lượng không hợp lệ: %q", s)
	}
	n, err := strconv.ParseFloat(t[:len(t)-1], 64)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("thời lượng không hợp lệ: %q", s)
	}
	unit := map[byte]time.Duration{
		's': time.Second, 'm': time.Minute, 'h': time.Hour,
		'd': 24 * time.Hour, 'w': 7 * 24 * time.Hour, 'y': 365 * 24 * time.Hour,
	}
	d, ok := unit[t[len(t)-1]]
	if !ok {
		return 0, fmt.Errorf("thời lượng không hợp lệ: %q", s)
	}
	return time.Duration(n * float64(d)), nil
}

// FormatBytes: 1536 → "1.5 KB".
func FormatBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	f, units := float64(n), []string{"KB", "MB", "GB", "TB"}
	i := -1
	for f >= 1024 && i < len(units)-1 {
		f /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", f, units[i])
}
