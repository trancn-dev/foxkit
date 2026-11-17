package cmd

import (
	"regexp"
	"strings"
	"testing"
)

func TestUUIDCmd(t *testing.T) {
	v4 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	v7 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	tests := []struct {
		name     string
		args     []string
		wantExit int
		re       *regexp.Regexp
		lines    int
	}{
		{"v4 default", []string{"uuid"}, 0, v4, 1},
		{"v7 sorted", []string{"uuid", "--v", "7", "--count", "50"}, 0, v7, 50},
		{"upper no-dash", []string{"uuid", "--upper", "--no-dash"}, 0, regexp.MustCompile(`^[0-9A-F]{32}$`), 1},
		{"bad version", []string{"uuid", "--v", "5"}, 2, nil, 0},
		{"bad count", []string{"uuid", "--count", "0"}, 2, nil, 0},
		{"bad count high", []string{"uuid", "--count", "1001"}, 2, nil, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, _, code := execute(tc.args...)
			if code != tc.wantExit {
				t.Fatalf("exit %d", code)
			}
			if tc.re == nil {
				return
			}
			lines := strings.Split(strings.TrimSpace(o), "\n")
			if len(lines) != tc.lines {
				t.Fatalf("lines %d want %d", len(lines), tc.lines)
			}
			for i, l := range lines {
				if !tc.re.MatchString(l) {
					t.Fatalf("%q không khớp", l)
				}
				if i > 0 && lines[i-1] > l {
					t.Fatalf("không tăng dần")
				}
			}
		})
	}
}

func TestUUIDUnique(t *testing.T) {
	o, _, _ := execute("uuid", "--count", "1000")
	seen := map[string]bool{}
	for _, l := range strings.Fields(o) {
		if seen[l] {
			t.Fatalf("trùng %s", l)
		}
		seen[l] = true
	}
}
