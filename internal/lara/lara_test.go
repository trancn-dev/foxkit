package lara

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func project(t *testing.T) string {
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "artisan"), nil, 0o644)
	os.WriteFile(filepath.Join(d, "composer.json"), []byte(`{"require":{"laravel/framework":"^11"}}`), 0o644)
	return d
}

func TestIsProject(t *testing.T) {
	if !IsProject(project(t)) || IsProject(t.TempDir()) {
		t.Fatal("IsProject sai")
	}
	d := t.TempDir()
	os.WriteFile(filepath.Join(d, "artisan"), nil, 0o644)
	if IsProject(d) {
		t.Error("thiếu composer.json không phải Laravel")
	}
}

func TestKeyIs32Bytes(t *testing.T) {
	k, err := NewKey()
	if err != nil || !strings.HasPrefix(k, "base64:") {
		t.Fatal(k, err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(k, "base64:"))
	if err != nil || len(raw) != 32 {
		t.Fatalf("phải đúng 32 byte: %d %v", len(raw), err)
	}
}

func TestApplyKey(t *testing.T) {
	d := project(t)
	envFile := filepath.Join(d, ".env")
	orig := "APP_NAME=Shop\r\nAPP_KEY=\r\nDB_HOST=127.0.0.1\r\n"
	os.WriteFile(envFile, []byte(orig), 0o644)
	if _, err := ApplyKey(d, "base64:AAA=", false); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(envFile)
	if string(got) != "APP_NAME=Shop\r\nAPP_KEY=base64:AAA=\r\nDB_HOST=127.0.0.1\r\n" {
		t.Fatalf("chỉ đúng dòng APP_KEY được đổi, giữ CRLF: %q", got)
	}
	if bak, _ := os.ReadFile(envFile + ".foxkit.bak"); string(bak) != orig {
		t.Error("backup phải là bản gốc")
	}
	if _, err := ApplyKey(d, "base64:BBB=", false); !errors.Is(err, ErrKeyExists) {
		t.Fatalf("đã có key không --force phải lỗi: %v", err)
	}
	if _, err := ApplyKey(d, "base64:BBB=", true); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(envFile); !strings.Contains(string(got), "APP_KEY=base64:BBB=") {
		t.Error("--force phải ghi đè")
	}
}

func TestClearFallbackForOldLaravel(t *testing.T) {
	old := artisan
	defer func() { artisan = old }()
	var calls []string
	artisan = func(_ context.Context, _ string, args ...string) ([]byte, int, error) {
		calls = append(calls, args[0])
		if args[0] == "optimize:clear" {
			return []byte(`Command "optimize:clear" is not defined.`), 1, nil
		}
		return []byte("ok\n"), 0, nil
	}
	r, err := Clear(context.Background(), "x")
	if err != nil || r.ExitCode != 0 || strings.Join(calls, ",") != "optimize:clear,config:clear,cache:clear,route:clear,view:clear" {
		t.Fatalf("%v %+v %v", calls, r, err)
	}
}

func TestClearPHPMissingAndFailure(t *testing.T) {
	old := artisan
	defer func() { artisan = old }()
	artisan = func(context.Context, string, ...string) ([]byte, int, error) { return nil, 0, ErrPHPNotFound }
	if _, err := Clear(context.Background(), "x"); !errors.Is(err, ErrPHPNotFound) {
		t.Fatalf("%v", err)
	}
	artisan = func(context.Context, string, ...string) ([]byte, int, error) { return []byte("lỗi khác"), 1, nil }
	if r, _ := Clear(context.Background(), "x"); r.ExitCode != 1 || len(r.Ran) != 1 {
		t.Fatalf("lỗi khác không được fallback: %+v", r)
	}
}

func TestLogFilesOrder(t *testing.T) {
	d := project(t)
	logs := filepath.Join(d, "storage", "logs")
	os.MkdirAll(logs, 0o755)
	for _, n := range []string{"laravel-2026-11-14.log", "laravel-2026-11-16.log", "laravel-2026-11-15.log", "other.log"} {
		os.WriteFile(filepath.Join(logs, n), nil, 0o644)
	}
	var base []string
	for _, p := range LogFiles(d) {
		base = append(base, filepath.Base(p))
	}
	if strings.Join(base, ",") != "laravel-2026-11-16.log,laravel-2026-11-15.log,laravel-2026-11-14.log" {
		t.Fatalf("%v", base)
	}
	os.WriteFile(filepath.Join(logs, "laravel.log"), nil, 0o644)
	if LogFiles(d)[0] != filepath.Join(logs, "laravel.log") {
		t.Error("laravel.log (single) phải đứng đầu")
	}
}
