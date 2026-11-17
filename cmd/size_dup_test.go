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

func TestRenameCmd(t *testing.T) {
	d := t.TempDir()
	for _, f := range []string{"IMG_1.JPG", "IMG_2.JPG"} {
		if err := os.WriteFile(filepath.Join(d, f), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name     string
		args     []string
		wantExit int
		out      string
	}{
		{"preview", []string{"rename", d, "--find", "IMG_", "--replace", "trip_"}, 0, "Đây là bản xem trước"},
		{"no mode", []string{"rename", d}, 2, ""},
		{"two modes", []string{"rename", d, "--find", "a", "--case", "lower"}, 2, ""},
		{"bad regex", []string{"rename", d, "--find", "(", "--regex"}, 2, ""},
		{"bad pattern", []string{"rename", d, "--number", "x"}, 2, ""},
		{"apply", []string{"rename", d, "--find", "IMG_", "--replace", "trip_", "--apply"}, 0, "Đã đổi tên 2 file"},
		{"apply strip prefix", []string{"rename", d, "--find", "trip_", "--replace", "", "--regex", "--apply", "--with-ext"}, 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, _, code := execute(tc.args...)
			if code != tc.wantExit || !strings.Contains(o, tc.out) {
				t.Fatalf("exit %d out %q", code, o)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(d, "1.JPG")); err != nil {
		t.Fatalf("lệnh cuối phải đổi trip_1.JPG → 1.JPG: %v", err)
	}
}
