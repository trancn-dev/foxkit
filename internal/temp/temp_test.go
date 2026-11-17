package temp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func put(t *testing.T, root, rel string, size int, age time.Duration) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := time.Now().Add(-age)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
	return p
}

const day = 24 * time.Hour

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func TestDryRunThenApply(t *testing.T) {
	d := t.TempDir()
	old := put(t, d, "old.tmp", 100, 10*day)
	fresh := put(t, d, "fresh.tmp", 50, day)
	r, err := Run(context.Background(), d, 7*day, false, time.Now())
	if err != nil || r.Candidates != 1 || r.CandidateBytes != 100 || r.TotalBytes != 150 || r.Deleted != 0 || !exists(old) {
		t.Fatalf("dry-run: %+v %v", r, err)
	}
	r, err = Run(context.Background(), d, 7*day, true, time.Now())
	if err != nil || r.Deleted != 1 || r.FreedBytes != 100 || exists(old) || !exists(fresh) {
		t.Fatalf("apply: %+v %v", r, err)
	}
}

func TestDirWithFreshFileIsKept(t *testing.T) {
	d := t.TempDir()
	put(t, d, "proj/old.txt", 10, 20*day)
	put(t, d, "proj/new.txt", 10, day)
	put(t, d, "stale/old.txt", 10, 20*day)
	r, err := Run(context.Background(), d, 7*day, true, time.Now())
	if err != nil || r.Deleted != 1 || !exists(filepath.Join(d, "proj", "new.txt")) || exists(filepath.Join(d, "stale")) {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestLockedFileIsCountedNotFatal(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows mới chặn xóa file đang mở")
	}
	d := t.TempDir()
	p := put(t, d, "busy.tmp", 10, 10*day)
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	r, err := Run(context.Background(), d, 7*day, true, time.Now())
	if err != nil || r.Locked != 1 || r.Deleted != 0 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestMissingDir(t *testing.T) {
	if _, err := Run(context.Background(), filepath.Join(t.TempDir(), "none"), day, false, time.Now()); err == nil {
		t.Fatal("cần lỗi")
	}
}
