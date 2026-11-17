package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSizeAndDupCmd(t *testing.T) {
	d := t.TempDir()
	for _, f := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(d, f), []byte(strings.Repeat("x", 2000)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name     string
		args     []string
		wantExit int
		out      string
	}{
		{"size", []string{"size", d}, 0, "a.txt"},
		{"size json", []string{"size", d, "--json", "--files", "--top", "1"}, 0, `"total": 4000`},
		{"size bad top", []string{"size", d, "--top", "0"}, 2, ""},
		{"size no dir", []string{"size", filepath.Join(d, "zz")}, 2, ""},
		{"dup", []string{"dup", d}, 0, "[2.0 KB × 2]"},
		{"dup json empty groups is []", []string{"dup", d, "--min-size", "1MB", "--json"}, 0, `"groups": []`},
		{"dup no dir", []string{"dup"}, 2, ""},
		{"dup bad size", []string{"dup", d, "--min-size", "zz"}, 2, ""},
		{"dup bad workers", []string{"dup", d, "--workers", "99"}, 2, ""},
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
