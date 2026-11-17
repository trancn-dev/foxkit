package lock

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestTypeName(t *testing.T) {
	for code, want := range map[uint32]string{0: "unknown", 1: "mainWindow", 2: "otherWindow", 3: "service", 4: "explorer", 5: "console", 1000: "critical", 7: "unknown"} {
		if TypeName(code) != want {
			t.Errorf("TypeName(%d) != %s", code, want)
		}
	}
}

func TestExpandDirLimitAndTruncated(t *testing.T) {
	d := t.TempDir()
	for i := 0; i < 10; i++ {
		os.WriteFile(filepath.Join(d, string(rune('a'+i))+".txt"), []byte("x"), 0o644)
	}
	files, truncated, err := Expand([]string{d}, 3)
	if err != nil || len(files) != 3 || !truncated {
		t.Fatalf("%d files, truncated=%v, err=%v", len(files), truncated, err)
	}
	files, truncated, _ = Expand([]string{d}, 50)
	if len(files) != 10 || truncated {
		t.Fatalf("%d files, truncated=%v", len(files), truncated)
	}
	one := filepath.Join(d, "a.txt")
	if files, _, _ := Expand([]string{one}, 1); len(files) != 1 || files[0] != one {
		t.Fatalf("file đơn: %v", files)
	}
	if _, _, err := Expand([]string{filepath.Join(d, "none")}, 1); err == nil {
		t.Error("đường dẫn không tồn tại phải lỗi")
	}
}

func TestBatches(t *testing.T) {
	b := Batches(make([]string, 2500), 1000)
	if len(b) != 3 || len(b[0]) != 1000 || len(b[2]) != 500 {
		t.Fatalf("%d lô", len(b))
	}
	if Batches(nil, 10) != nil {
		t.Error("rỗng → nil")
	}
}

// Tự khóa: mở file trong chính test rồi hỏi Restart Manager; PID phải là của test. Đóng file → rỗng.
func TestFindSelfHolder(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ Windows")
	}
	p := filepath.Join(t.TempDir(), "locked.txt")
	f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Find([]string{p}, 10)
	if err != nil {
		f.Close()
		t.Fatal(err)
	}
	found := false
	for _, h := range res.Holders {
		if int(h.PID) == os.Getpid() {
			found = true
			if h.Name == "" {
				t.Errorf("thiếu tên: %+v", h)
			}
		}
	}
	if !found {
		f.Close()
		t.Fatalf("PID %d phải nằm trong holders: %+v", os.Getpid(), res.Holders)
	}
	f.Close()
	res, err = Find([]string{p}, 10)
	if err != nil || len(res.Holders) != 0 || res.Holders == nil {
		t.Fatalf("sau khi đóng phải rỗng ([] không nil): %+v %v", res, err)
	}
}
