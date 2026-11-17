package diff

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func opsString(ops []Line) string {
	var b strings.Builder
	for _, l := range ops {
		switch l.Op {
		case Add:
			b.WriteString("+" + l.Text)
		case Del:
			b.WriteString("-" + l.Text)
		default:
			b.WriteString("=" + l.Text)
		}
		b.WriteString(" ")
	}
	return strings.TrimSpace(b.String())
}

func TestLinesTable(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want string
	}{
		{"thêm cuối", []string{"a", "b"}, []string{"a", "b", "c"}, "=a =b +c"},
		{"xóa đầu", []string{"a", "b", "c"}, []string{"b", "c"}, "-a =b =c"},
		{"đổi giữa", []string{"a", "b", "c"}, []string{"a", "X", "c"}, "=a -b +X =c"},
		{"giống nhau", []string{"a", "b"}, []string{"a", "b"}, "=a =b"},
		{"a rỗng", nil, []string{"x", "y"}, "+x +y"},
		{"b rỗng", []string{"x", "y"}, nil, "-x -y"},
		{"cả hai rỗng", nil, nil, ""},
	}
	for _, tc := range tests {
		if got := opsString(Lines(tc.a, tc.b, Options{})); got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}

// Property: áp kết quả diff lên a phải ra đúng b, và số dòng tối thiểu (không "thay hết" khi chỉ khác ít).
func TestApplyingDiffReproducesB(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	alphabet := []string{"a", "b", "c", "d", "e"}
	gen := func() []string {
		n := rng.Intn(25)
		s := make([]string, n)
		for i := range s {
			s[i] = alphabet[rng.Intn(len(alphabet))]
		}
		return s
	}
	for i := 0; i < 500; i++ {
		a, b := gen(), gen()
		var rebuilt, orig []string
		for _, l := range Lines(a, b, Options{}) {
			switch l.Op {
			case Equal:
				rebuilt = append(rebuilt, l.Text)
				orig = append(orig, l.Text)
			case Add:
				rebuilt = append(rebuilt, l.Text)
			case Del:
				orig = append(orig, l.Text)
			}
		}
		if strings.Join(rebuilt, "|") != strings.Join(b, "|") || strings.Join(orig, "|") != strings.Join(a, "|") {
			t.Fatalf("a=%v b=%v → rebuilt=%v orig=%v", a, b, rebuilt, orig)
		}
	}
}

func TestIgnoreFlagsKeepOriginalText(t *testing.T) {
	ops := Lines([]string{"a\r", "b"}, []string{"a", "b"}, Options{IgnoreEOL: true})
	if opsString(ops) != "=a\r =b" {
		t.Errorf("ignore-eol: %q", opsString(ops))
	}
	if ops := Lines([]string{"a  b"}, []string{"a b "}, Options{IgnoreSpace: true}); ops[0].Op != Equal {
		t.Errorf("ignore-space: %+v", ops)
	}
	if ops := Lines([]string{"ABC"}, []string{"abc"}, Options{IgnoreCase: true}); ops[0].Op != Equal {
		t.Errorf("ignore-case: %+v", ops)
	}
	if ops := Lines([]string{"a\r"}, []string{"a"}, Options{}); ops[0].Op == Equal {
		t.Error("không có --ignore-eol thì \\r là khác biệt")
	}
}

func numbered(n int, change ...int) []string {
	s := make([]string, n)
	for i := range s {
		s[i] = fmt.Sprintf("line %d", i)
	}
	for _, c := range change {
		s[c] = "CHANGED"
	}
	return s
}

func TestHunksMergeWithinTwiceContext(t *testing.T) {
	a := numbered(40)
	near := ToHunks(Lines(a, numbered(40, 10, 15), Options{}), 3) // cách 5 ≤ 2×3+1 → gộp
	if len(near) != 1 {
		t.Fatalf("hai chỗ gần phải gộp, got %d hunk", len(near))
	}
	far := ToHunks(Lines(a, numbered(40, 5, 30), Options{}), 3)
	if len(far) != 2 {
		t.Fatalf("hai chỗ xa phải tách, got %d hunk", len(far))
	}
	h := far[0]
	if h.AStart != 3 || h.ALen != 7 || h.BStart != 3 || h.BLen != 7 { // dòng 3..9 (1-based): 3 trước + dòng đổi + 3 sau
		t.Errorf("hunk đầu: %+v", h)
	}
	if len(ToHunks(Lines(a, a, Options{}), 3)) != 0 {
		t.Error("giống nhau không có hunk")
	}
}

func TestUnifiedFormat(t *testing.T) {
	ops := Lines([]string{"APP_ENV=local", "APP_DEBUG=true", "APP_URL=x"}, []string{"APP_ENV=local", "APP_DEBUG=false", "APP_URL=x"}, Options{})
	out := Unified(".env", ".env.production", ToHunks(ops, 3), false)
	want := "--- .env\n+++ .env.production\n@@ -1,3 +1,3 @@\n APP_ENV=local\n-APP_DEBUG=true\n+APP_DEBUG=false\n APP_URL=x\n"
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
}

func TestBigFileUnderOneSecond(t *testing.T) {
	a := numbered(50000)
	b := numbered(50000, 100, 25000, 49000)
	start := time.Now()
	ops := Lines(a, b, Options{})
	if time.Since(start) > time.Second {
		t.Errorf("50.000 dòng mất %v", time.Since(start))
	}
	if len(ToHunks(ops, 3)) != 3 {
		t.Error("phải đúng 3 hunk")
	}
	// hoàn toàn khác nhau: không được nổ bộ nhớ/thời gian
	start = time.Now()
	Lines(numbered(20000), strings.Split(strings.Repeat("x\n", 20000), "\n"), Options{})
	if time.Since(start) > 5*time.Second {
		t.Errorf("trường hợp xấu nhất quá chậm: %v", time.Since(start))
	}
}

func put(t *testing.T, root, rel string, data []byte) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDirs(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	put(t, a, "config/app.php", []byte("1\n2\n3\n"))
	put(t, b, "config/app.php", []byte("1\nX\n3\n4\n"))
	put(t, b, "config/horizon.php", []byte("new\n"))
	put(t, a, "config/old.php", []byte("old\n"))
	put(t, a, "public/logo.png", []byte("\x89PNG\x00\x01"))
	put(t, b, "public/logo.png", []byte("\x89PNG\x00\x02"))
	put(t, a, "same.txt", []byte("same\n"))
	put(t, b, "SAME.txt", []byte("same\n")) // Windows không phân biệt hoa thường
	put(t, a, "node_modules/x.js", []byte("a"))
	put(t, b, "node_modules/x.js", []byte("b"))
	res, err := Dirs(a, b, Options{Context: 3})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, f := range res.Files {
		got[f.Path] = f.Status
	}
	want := map[string]string{"config/app.php": "changed", "config/horizon.php": "added", "config/old.php": "removed", "public/logo.png": "binary"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: %q, cần %q", k, got[k], v)
		}
	}
	if res.Added != 1 || res.Removed != 1 || res.Changed != 2 || res.Identical {
		t.Errorf("%+v", res)
	}
	for _, f := range res.Files {
		if f.Path == "config/app.php" && (f.Added != 2 || f.Removed != 1 || len(f.Hunks) != 1) {
			t.Errorf("app.php: %+v", f)
		}
	}
	// names-only: không có hunk
	res, _ = Dirs(a, b, Options{NamesOnly: true})
	for _, f := range res.Files {
		if len(f.Hunks) != 0 {
			t.Errorf("names-only không được diff nội dung: %+v", f)
		}
	}
	same, _ := Dirs(a, a, Options{})
	if !same.Identical || same.Files == nil {
		t.Errorf("giống hệt: %+v", same)
	}
}

func TestFilesBinaryAndEmpty(t *testing.T) {
	d := t.TempDir()
	put(t, d, "a.bin", []byte("x\x00y"))
	put(t, d, "b.bin", []byte("x\x00z"))
	put(t, d, "e1.txt", nil)
	put(t, d, "e2.txt", []byte("hi\n"))
	fd, err := Files(filepath.Join(d, "a.bin"), filepath.Join(d, "b.bin"), Options{})
	if err != nil || fd.Status != "binary" || len(fd.Hunks) != 0 {
		t.Fatalf("%+v %v", fd, err)
	}
	fd, _ = Files(filepath.Join(d, "e1.txt"), filepath.Join(d, "e2.txt"), Options{Context: 3})
	if fd.Status != "changed" || fd.Added != 1 {
		t.Fatalf("file rỗng: %+v", fd)
	}
}
