package cmd

import (
	"strings"
	"testing"
)

func TestEncCmd(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantExit int
		out      string
	}{
		{"encode", []string{"enc", "base64", "hello"}, 0, "aGVsbG8="},
		{"decode", []string{"enc", "base64", "-d", "aGVsbG8"}, 0, "hello"},
		{"unicode", []string{"enc", "base64", "Xin chào"}, 0, "WGluIGNow6Bv"},
		{"json", []string{"enc", "hex", "hi", "--json"}, 0, `"result": "6869"`},
		{"binary json", []string{"enc", "hex", "-d", "ff", "--json"}, 0, `"binary": true`},
		{"bad mode", []string{"enc", "zz", "x"}, 2, ""},
		{"no input", []string{"enc", "base64"}, 2, ""},
		{"decode fail", []string{"enc", "base64", "-d", "!!!"}, 1, ""},
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
