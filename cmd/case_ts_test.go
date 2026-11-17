package cmd

import (
	"strings"
	"testing"
)

func TestCaseAndTsCmd(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		out      string
	}{
		{"snake", []string{"case", "snake", "userName"}, 0, "user_name"},
		{"all", []string{"case", "all", "Đơn hàng mới"}, 0, "camel     donHangMoi"},
		{"json", []string{"case", "slug", "Đơn hàng mới", "--json"}, 0, `"result": "don-hang-moi"`},
		{"bad mode", []string{"case", "zz", "x"}, 2, ""},
		{"no input", []string{"case", "snake"}, 2, ""},
		{"ts seconds", []string{"ts", "0", "--tz", "UTC"}, 0, "1970-01-01T00:00:00Z"},
		{"ts json detected", []string{"ts", "1763287727000", "--tz", "UTC", "--json"}, 0, `"detected": "milliseconds"`},
		{"ts now", []string{"ts", "--tz", "Asia/Ho_Chi_Minh"}, 0, "Asia/Ho_Chi_Minh (+07:00)"},
		{"ts bad", []string{"ts", "abc"}, 2, ""},
		{"ts bad tz", []string{"ts", "0", "--tz", "Mars/Base"}, 2, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, _, code := execute(tc.args...)
			if code != tc.wantExit || !strings.Contains(o, tc.out) {
				t.Fatalf("exit %d out %q", code, o)
			}
		})
	}
}
