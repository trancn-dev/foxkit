package find

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
)

func put(t *testing.T, root, rel string, data []byte) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func names(r Result) string {
	var n []string
	for _, h := range r.Hits {
		n = append(n, h.Name)
	}
	return strings.Join(n, ",")
}

func run(t *testing.T, o Options) Result {
	t.Helper()
	r, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestNameMatching(t *testing.T) {
	d := t.TempDir()
	put(t, d, "Hợp Đồng.pdf", []byte("x"))
	put(t, d, "a.pdf", []byte("x"))
	put(t, d, "b.txt", []byte("x"))
	if got := names(run(t, Options{Dir: d, Pattern: "hop dong"})); got != "Hợp Đồng.pdf" {
		t.Errorf("không dấu: %q", got)
	}
	if got := names(run(t, Options{Dir: d, Pattern: "*.PDF"})); got != "Hợp Đồng.pdf,a.pdf" && got != "a.pdf,Hợp Đồng.pdf" {
		t.Errorf("glob: %q", got)
	}
	if got := names(run(t, Options{Dir: d, Exts: []string{".txt"}})); got != "b.txt" {
		t.Errorf("ext: %q", got)
	}
	if ValidPattern("[abc") || !ValidPattern("*.pdf") {
		t.Error("ValidPattern sai")
	}
}

func TestSizeAndAge(t *testing.T) {
	d := t.TempDir()
	put(t, d, "small", make([]byte, 500))
	put(t, d, "mid", make([]byte, 2000))
	put(t, d, "big", make([]byte, 2<<20))
	k, m := int64(1024), int64(1<<20)
	if got := names(run(t, Options{Dir: d, Larger: &k, Smaller: &m})); got != "mid" {
		t.Errorf("size: %q", got)
	}
	old := time.Now().Add(-10 * 24 * time.Hour)
	if err := os.Chtimes(filepath.Join(d, "small"), old, old); err != nil {
		t.Fatal(err)
	}
	if got := names(run(t, Options{Dir: d, Newer: 7 * 24 * time.Hour})); strings.Contains(got, "small") || !strings.Contains(got, "mid") {
		t.Errorf("newer: %q", got)
	}
	if got := names(run(t, Options{Dir: d, Older: 7 * 24 * time.Hour})); got != "small" {
		t.Errorf("older: %q", got)
	}
}

func TestContent(t *testing.T) {
	d := t.TempDir()
	put(t, d, "a.txt", []byte("dòng 1\nRoute::post('/Coupon')\n"))
	put(t, d, "bin.dat", []byte("xx\x00Coupon"))
	u := utf16.Encode([]rune("dòng\r\nChứa Coupon ở đây\r\n"))
	b := []byte{0xFF, 0xFE}
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	put(t, d, "w.reg", b)
	r := run(t, Options{Dir: d, Content: "coupon", IgnoreCase: true})
	if names(r) != "a.txt,w.reg" {
		t.Fatalf("hits: %q", names(r))
	}
	if r.Hits[0].Line != 2 || !strings.Contains(r.Hits[0].Snippet, "Route::post") || r.Hits[1].Line != 2 {
		t.Fatalf("%+v", r.Hits)
	}
	if got := run(t, Options{Dir: d, Content: "coupon"}); len(got.Hits) != 0 {
		t.Errorf("phân biệt hoa thường khi không bật ignore-case: %q", names(got))
	}
}

func TestMaxStopsEarlyAndTruncates(t *testing.T) {
	d := t.TempDir()
	for _, n := range []string{"a", "b", "c", "d", "e", "f"} {
		put(t, d, n+".txt", []byte("x"))
	}
	r := run(t, Options{Dir: d, Max: 3})
	if len(r.Hits) != 3 || !r.Truncated || r.Scanned > 3 {
		t.Fatalf("%+v", r)
	}
	r = run(t, Options{Dir: d, Max: 2, Content: "x"})
	if len(r.Hits) != 2 || !r.Truncated {
		t.Fatalf("content max: %+v", r)
	}
}

func TestSkipDirsAndTypes(t *testing.T) {
	d := t.TempDir()
	put(t, d, "node_modules/x.js", []byte("x"))
	put(t, d, "sub/y.js", []byte("x"))
	if got := names(run(t, Options{Dir: d, Pattern: ".js"})); got != "y.js" {
		t.Errorf("skip node_modules: %q", got)
	}
	if got := names(run(t, Options{Dir: d, Hidden: true, Pattern: ".js"})); !strings.Contains(got, "x.js") {
		t.Errorf("--hidden: %q", got)
	}
	if got := names(run(t, Options{Dir: d, Type: "d"})); got != "sub" {
		t.Errorf("type d: %q", got)
	}
	if got := run(t, Options{Dir: d, Pattern: "zzz"}); got.Hits == nil || len(got.Hits) != 0 {
		t.Error("Hits phải là [] không nil")
	}
}
