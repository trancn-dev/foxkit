package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvCmd(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".env.example", "APP_KEY=base64:x\nDB_HOST=localhost\nREDIS_HOST=127.0.0.1\n")
	write(".env", "APP_KEY=\nDB_HOST=127.0.0.1\nOLD=1\nAPP_SECRET=abcdefghijk\n")
	tests := []struct {
		name     string
		args     []string
		wantExit int
		out      string
	}{
		{"diff", []string{"env", "diff", dir}, 0, "Thiếu (1):  REDIS_HOST"},
		{"diff json", []string{"env", "diff", dir, "--json"}, 0, `"extra": [`},
		{"get plain", []string{"env", "get", "DB_HOST", dir}, 0, "127.0.0.1"},
		{"get masked", []string{"env", "get", "APP_SECRET", dir}, 0, "abc***"},
		{"get show", []string{"env", "get", "APP_SECRET", dir, "--show"}, 0, "abcdefghijk"},
		{"get missing", []string{"env", "get", "NOPE", dir}, 1, ""},
		{"no dir", []string{"env", "diff", filepath.Join(dir, "zz")}, 2, ""},
		{"no env file", []string{"env", "diff", t.TempDir()}, 1, ""},
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

func TestPortCmdArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		exit int
	}{
		{"zero", []string{"port", "0"}, 2},
		{"high", []string{"port", "70000"}, 2},
		{"abc", []string{"port", "abc"}, 2},
		{"missing", []string{"port"}, 2},
		{"list+port", []string{"port", "80", "--list"}, 2},
		{"list+kill", []string{"port", "--list", "--kill"}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, code := execute(tc.args...); code != tc.exit {
				t.Fatalf("exit %d", code)
			}
		})
	}
}
