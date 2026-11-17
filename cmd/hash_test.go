package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHashCmd(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(f, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	const abc = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	tests := []struct {
		name     string
		args     []string
		wantExit int
		inOut    string
	}{
		{"text", []string{"hash", "--text", "abc"}, 0, abc},
		{"file", []string{"hash", f}, 0, abc + "  " + f},
		{"check upper", []string{"hash", f, "--check", strings.ToUpper(abc)}, 0, "khớp: true"},
		{"check bad", []string{"hash", f, "--check", "00"}, 1, "khớp: false"},
		{"all", []string{"hash", "--text", "abc", "--all"}, 0, "crc32  352441c2"},
		{"missing file keeps going", []string{"hash", filepath.Join(dir, "none"), f}, 1, abc},
		{"no input", []string{"hash"}, 2, ""},
		{"conflict", []string{"hash", f, "--text", "x"}, 2, ""},
		{"check many", []string{"hash", f, f, "--check", "x"}, 2, ""},
		{"bad algo", []string{"hash", f, "--algo", "zz"}, 2, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o, _, code := execute(tc.args...)
			if code != tc.wantExit || !strings.Contains(o, tc.inOut) {
				t.Fatalf("exit %d out %q", code, o)
			}
		})
	}
}
