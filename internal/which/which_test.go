package which

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func rel(root string, ms []Match) string {
	var out []string
	for _, m := range ms {
		r, _ := filepath.Rel(root, m.Path)
		out = append(out, filepath.ToSlash(r))
	}
	return strings.Join(out, ",")
}

func TestFindOrderAndActive(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	touch(t, filepath.Join(a, "php.exe"))
	touch(t, filepath.Join(b, "php.exe"))
	touch(t, filepath.Join(b, "php.bat"))
	path := a + ";" + b
	ms := Find("php", path, ".COM;.EXE;.BAT;.CMD", "")
	if got := rel(root, ms); got != "a/php.exe,b/php.exe,b/php.bat" || !ms[0].Active || ms[1].Active {
		t.Fatalf("got %s", got)
	}
	if got := rel(root, Find("php.EXE", path, "", "")); got != "a/php.exe,b/php.exe" {
		t.Fatalf("đuôi viết hoa vẫn phải khớp: %s", got)
	}
}

func TestDedupeAndCwdFirst(t *testing.T) {
	root := t.TempDir()
	a, cwd := filepath.Join(root, "a"), filepath.Join(root, "cwd")
	touch(t, filepath.Join(a, "tool.exe"))
	touch(t, filepath.Join(cwd, "tool.exe"))
	ms := Find("tool", a+";"+strings.ToUpper(a)+";"+a, "", cwd)
	if got := rel(root, ms); got != "cwd/tool.exe,a/tool.exe" {
		t.Fatalf("cwd trước, PATH lặp không trùng: %s", got)
	}
}

func TestNotFoundIsEmptySliceAndDirIgnored(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "x.exe"), 0o755); err != nil { // thư mục tên x.exe không phải file chạy được
		t.Fatal(err)
	}
	ms := Find("x", root, "", "")
	if ms == nil || len(ms) != 0 {
		t.Fatalf("%v", ms)
	}
}
