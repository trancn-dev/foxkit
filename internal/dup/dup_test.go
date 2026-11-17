package dup

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func put(t *testing.T, root, rel, content string) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func rels(root string, g Group) []string {
	var out []string
	for _, f := range g.Files {
		r, _ := filepath.Rel(root, f)
		out = append(out, filepath.ToSlash(r))
	}
	return out
}

func TestFindBasic(t *testing.T) {
	d := t.TempDir()
	put(t, d, "a.txt", "hello")
	put(t, d, "b.txt", "hello")
	put(t, d, "c.txt", "hellp")              // cùng size, khác nội dung
	put(t, d, "node_modules/x.txt", "hello") // bị bỏ qua
	r, err := Find(context.Background(), Options{Dirs: []string{d}, Workers: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Groups) != 1 || !reflect.DeepEqual(rels(d, r.Groups[0]), []string{"a.txt", "b.txt"}) || r.WastedBytes != 5 {
		t.Fatalf("%+v", r)
	}
}

func TestSame4KBHeadDifferentTail(t *testing.T) {
	d := t.TempDir()
	head := strings.Repeat("A", 4096)
	put(t, d, "one.bin", head+"tail-1")
	put(t, d, "two.dat", head+"tail-2")
	r, err := Find(context.Background(), Options{Dirs: []string{d}, Workers: 2})
	if err != nil || len(r.Groups) != 0 || r.Groups == nil {
		t.Fatalf("không được coi là trùng (và Groups phải là [] không nil): %+v %v", r, err)
	}
}

func TestExtAndMinSize(t *testing.T) {
	d := t.TempDir()
	put(t, d, "a.txt", "hello")
	put(t, d, "b.txt", "hello")
	put(t, d, "a.log", "hello")
	put(t, d, "b.log", "hello")
	r, _ := Find(context.Background(), Options{Dirs: []string{d}, Exts: []string{".TXT"}, Workers: 1})
	if len(r.Groups) != 1 || len(r.Groups[0].Files) != 2 || r.Scanned != 2 {
		t.Fatalf("lọc ext sai: %+v", r)
	}
	r, _ = Find(context.Background(), Options{Dirs: []string{d}, MinSize: 10, Workers: 1})
	if len(r.Groups) != 0 || r.Scanned != 0 {
		t.Fatalf("min-size sai: %+v", r)
	}
}

func TestNestedDirsCountOnce(t *testing.T) {
	d := t.TempDir()
	put(t, d, "sub/a.txt", "hello")
	r, _ := Find(context.Background(), Options{Dirs: []string{d, filepath.Join(d, "sub")}, Workers: 1})
	if r.Scanned != 1 || len(r.Groups) != 0 {
		t.Fatalf("một file liệt kê hai lần phải khử trùng: %+v", r)
	}
}

func TestWorkersGiveSameResult(t *testing.T) {
	d := t.TempDir()
	for i := 0; i < 30; i++ {
		put(t, d, filepath.Join("d", string(rune('a'+i%6)), "f"+string(rune('A'+i))+".txt"), strings.Repeat("z", 10+i%4))
	}
	a, err1 := Find(context.Background(), Options{Dirs: []string{d}, Workers: 1})
	b, err2 := Find(context.Background(), Options{Dirs: []string{d}, Workers: 8})
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	a.DurationMs, b.DurationMs = 0, 0
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("khác nhau:\n%+v\n%+v", a, b)
	}
}

func TestCanceled(t *testing.T) {
	d := t.TempDir()
	put(t, d, "a.txt", "hello")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Find(ctx, Options{Dirs: []string{d}}); err == nil {
		t.Fatal("cần lỗi")
	}
}
