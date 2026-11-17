package syncdir

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func put(t *testing.T, root, rel, body string, mt time.Time) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if !mt.IsZero() {
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}
}

func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || p == root || strings.Contains(p, trashDir) {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		if info.IsDir() {
			out[filepath.ToSlash(rel)] = "dir"
			return nil
		}
		b, _ := os.ReadFile(p)
		h := sha256.Sum256(b)
		out[filepath.ToSlash(rel)] = string(h[:6])
		return nil
	})
	return out
}

func run(t *testing.T, src, dst string, opt Options) Plan {
	t.Helper()
	p, err := Build(context.Background(), src, dst, opt)
	if err != nil {
		t.Fatal(err)
	}
	if err := Apply(context.Background(), &p, opt, nil); err != nil {
		t.Fatal(err)
	}
	return p
}

var base = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

func trees(t *testing.T) (string, string) {
	a, b := t.TempDir(), filepath.Join(t.TempDir(), "backup")
	put(t, a, "a.txt", "hello", base)
	put(t, a, "sub/b.txt", "world", base)
	put(t, a, "Đơn hàng/c.txt", "xin chào", base)
	os.MkdirAll(filepath.Join(a, "empty"), 0o755)
	return a, b
}

func TestEmptyDestCopiesAllThenIsIdempotent(t *testing.T) {
	a, b := trees(t)
	p := run(t, a, b, Options{})
	if len(p.Failed) != 0 || !p.Applied {
		t.Fatalf("%+v", p)
	}
	wa, wb := snapshot(t, a), snapshot(t, b)
	if len(wa) != len(wb) {
		t.Fatalf("khác nhau:\n%v\n%v", wa, wb)
	}
	for k, v := range wa {
		if wb[k] != v {
			t.Errorf("%s khác", k)
		}
	}
	again, err := Build(context.Background(), a, b, Options{})
	if err != nil || len(again.Actions) != 0 || again.Same != 3 {
		t.Fatalf("chạy lại phải rỗng (mtime được giữ): %+v %v", again, err)
	}
}

func TestUpdateDetectionAndHashMode(t *testing.T) {
	a, b := trees(t)
	run(t, a, b, Options{})
	put(t, a, "a.txt", "HELLO", base.Add(time.Hour)) // cùng size, đổi nội dung + mtime
	p, _ := Build(context.Background(), a, b, Options{})
	if len(p.Actions) != 1 || p.Actions[0].Op != "update" || p.Actions[0].Reason != "mtime khác" {
		t.Fatalf("%+v", p.Actions)
	}
	// hash bắt được cả khi mtime không đổi
	run(t, a, b, Options{})
	put(t, a, "a.txt", "HELLX", base.Add(time.Hour)) // cùng size, cùng mtime (đã bằng nhau), khác nội dung
	if p, _ := Build(context.Background(), a, b, Options{}); len(p.Actions) != 0 {
		t.Fatalf("size+mtime không thấy khác biệt: %+v", p.Actions)
	}
	p, _ = Build(context.Background(), a, b, Options{Compare: "hash"})
	if len(p.Actions) != 1 || p.Actions[0].Reason != "hash khác" {
		t.Fatalf("--compare hash phải bắt được: %+v", p.Actions)
	}
}

func TestMtimeSkewWithinTwoSecondsIsSame(t *testing.T) {
	a, b := trees(t)
	run(t, a, b, Options{})
	put(t, b, "a.txt", "hello", base.Add(1*time.Second)) // giả FAT: lệch 1 giây
	if p, _ := Build(context.Background(), a, b, Options{}); len(p.Actions) != 0 {
		t.Fatalf("lệch 1s phải coi là giống: %+v", p.Actions)
	}
	put(t, b, "a.txt", "hello", base.Add(10*time.Second))
	if p, _ := Build(context.Background(), a, b, Options{}); len(p.Actions) != 1 {
		t.Fatalf("lệch 10s phải khác: %+v", p.Actions)
	}
}

func TestExtraFilesKeptUnlessDelete(t *testing.T) {
	a, b := trees(t)
	run(t, a, b, Options{})
	put(t, b, "extra.txt", "thừa", base)
	put(t, b, "olddir/x.txt", "x", base)
	p, _ := Build(context.Background(), a, b, Options{})
	if len(p.Actions) != 0 || p.OnlyInDst != 3 { // extra.txt, olddir, olddir/x.txt
		t.Fatalf("không --delete thì không đụng: %+v", p)
	}
	fixed := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	opt := Options{Delete: true, Now: func() time.Time { return fixed }}
	p = run(t, a, b, opt)
	if _, err := os.Stat(filepath.Join(b, "extra.txt")); err == nil {
		t.Error("extra.txt phải bị chuyển đi")
	}
	trash := filepath.Join(b, trashDir, "20260505-100000")
	if got, _ := os.ReadFile(filepath.Join(trash, "extra.txt")); string(got) != "thừa" {
		t.Errorf("phải nằm trong thùng rác, không mất: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(trash, "olddir", "x.txt")); string(got) != "x" {
		t.Error("thư mục thừa chuyển cả cây")
	}
	if again, _ := Build(context.Background(), a, b, Options{Delete: true}); len(again.Actions) != 0 {
		t.Errorf("thùng rác phải tự bị loại khỏi so sánh: %+v", again.Actions)
	}
}

func TestSafetyChecks(t *testing.T) {
	a := t.TempDir()
	if _, err := Build(context.Background(), a, filepath.Join(a, "inside"), Options{}); !errors.Is(err, ErrOverlap) {
		t.Errorf("dst trong src: %v", err)
	}
	if _, err := Build(context.Background(), filepath.Join(a, "x"), a, Options{}); err == nil {
		t.Error("src không tồn tại phải lỗi")
	}
	for _, bad := range []string{`C:\`, `d:\`, `C:\Windows`, `C:\Program Files`, `C:\Users`} {
		if !UnsafeDst(bad) {
			t.Errorf("%q phải là đích không an toàn", bad)
		}
	}
	if UnsafeDst(`E:\Backup\Projects`) {
		t.Error("thư mục con riêng là an toàn")
	}
	if _, err := Build(context.Background(), a, `C:\Windows`, Options{}); !errors.Is(err, ErrDstUnsafe) {
		t.Errorf("%v", err)
	}
}

func TestDefaultAndUserExcludes(t *testing.T) {
	a, b := t.TempDir(), filepath.Join(t.TempDir(), "bk")
	put(t, a, "keep.txt", "k", base)
	put(t, a, "Thumbs.db", "x", base)
	put(t, a, "tmp.tmp", "x", base)
	put(t, a, "node_modules/p/i.js", "x", base)
	p := run(t, a, b, Options{})
	got := snapshot(t, b)
	if _, ok := got["Thumbs.db"]; ok || got["tmp.tmp"] != "" {
		t.Errorf("mặc định phải loại Thumbs.db và *.tmp: %v", got)
	}
	if _, ok := got["node_modules/p/i.js"]; !ok {
		t.Errorf("node_modules mặc định được chép (backup phải đủ): %v", got)
	}
	_ = p
	b2 := filepath.Join(t.TempDir(), "bk2")
	run(t, a, b2, Options{Exclude: []string{"node_modules"}})
	if _, ok := snapshot(t, b2)["node_modules"]; ok {
		t.Error("--exclude node_modules phải bỏ được")
	}
}

func TestNoLeftoverTmpOnCancel(t *testing.T) {
	a, b := trees(t)
	put(t, a, "big.bin", strings.Repeat("x", 1<<20), base)
	p, err := Build(context.Background(), a, b, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Apply(ctx, &p, Options{}, nil); err == nil {
		t.Error("hủy phải trả lỗi")
	}
	filepath.Walk(b, func(path string, info os.FileInfo, err error) error {
		if err == nil && strings.HasSuffix(path, tmpExt) {
			t.Errorf("còn file tạm: %s", path)
		}
		return nil
	})
}

func TestLockedSourceFileFailsOthersCopied(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("khóa độc quyền kiểu Windows")
	}
	a, b := trees(t)
	locked := filepath.Join(a, "sub", "b.txt")
	ptr, _ := windows.UTF16PtrFromString(locked)
	h, err := windows.CreateFile(ptr, windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0) // shareMode 0 = độc quyền
	if err != nil {
		t.Skip("không khóa được file:", err)
	}
	defer windows.CloseHandle(h)
	p := run(t, a, b, Options{})
	if len(p.Failed) != 1 || !strings.Contains(p.Failed[0].Path, "b.txt") {
		t.Fatalf("file bị khóa phải vào Failed: %+v", p.Failed)
	}
	if got, _ := os.ReadFile(filepath.Join(b, "a.txt")); string(got) != "hello" {
		t.Error("các file khác vẫn phải được chép")
	}
}
