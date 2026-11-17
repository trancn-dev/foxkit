package winpath

import (
	"os"
	"path/filepath"
	"strings"
)

type Scope string

const (
	User   Scope = "user"
	System Scope = "system"
)

// Raw: giá trị PATH trong registry; Expand = kiểu REG_EXPAND_SZ.
type Raw struct {
	Value  string
	Expand bool
}

// splitRaw tách theo ";" và giữ nguyên từng mục gốc (kể cả %USERPROFILE%\...), bỏ mục rỗng.
func splitRaw(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ";") {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

// key: dạng chuẩn hóa chỉ để so sánh (expand %VAR% → Clean → bỏ "\" cuối → chữ thường); không bao giờ được ghi lại.
func key(entry string) string { return Normalize(Expand(strings.TrimSpace(entry))) }

func join(parts []string) string { return strings.Join(parts, ";") }

// Add thêm dir vào cuối (hoặc đầu nếu first); đã có (so theo dạng chuẩn hóa) thì giữ nguyên, changed=false.
func Add(value, dir string, first bool) (newValue string, changed bool) {
	parts := splitRaw(value)
	k := key(dir)
	for _, p := range parts {
		if key(p) == k {
			return join(parts), false
		}
	}
	if first {
		parts = append([]string{dir}, parts...)
	} else {
		parts = append(parts, dir)
	}
	return join(parts), true
}

// Remove xóa mọi mục khớp dir (so theo dạng chuẩn hóa).
func Remove(value, dir string) (newValue string, removed int) {
	k := key(dir)
	var keep []string
	for _, p := range splitRaw(value) {
		if key(p) == k {
			removed++
			continue
		}
		keep = append(keep, p)
	}
	return join(keep), removed
}

// Clean xóa mục rỗng, không tồn tại và trùng (giữ lần xuất hiện đầu). exists nhận đường dẫn đã expand.
func Clean(value string, exists func(string) bool) (newValue string, removed []string) {
	seen := map[string]bool{}
	var keep []string
	for _, raw := range strings.Split(value, ";") {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		k := key(raw)
		if seen[k] || !exists(Expand(strings.TrimSpace(raw))) {
			removed = append(removed, raw)
			continue
		}
		seen[k] = true
		keep = append(keep, raw)
	}
	return join(keep), removed
}

// Portable: dir nằm dưới %USERPROFILE% thì ghi lại dạng %USERPROFILE%\... để PATH dùng được khi đổi tên user (chỉ dùng cho scope user).
func Portable(dir string) string {
	up := os.Getenv("USERPROFILE")
	if up == "" {
		return dir
	}
	d, u := filepath.Clean(dir), filepath.Clean(up)
	if strings.EqualFold(d, u) {
		return "%USERPROFILE%"
	}
	if len(d) > len(u) && strings.EqualFold(d[:len(u)], u) && d[len(u)] == filepath.Separator {
		return "%USERPROFILE%" + d[len(u):]
	}
	return dir
}

// NeedsExpand: giá trị có %VAR% thì phải ghi kiểu REG_EXPAND_SZ (ghi SZ làm biến không còn được expand → hỏng PATH).
func NeedsExpand(r Raw) bool { return r.Expand || strings.Contains(r.Value, "%") }

const (
	WarnLength = 2047
	MaxLength  = 32767
)
