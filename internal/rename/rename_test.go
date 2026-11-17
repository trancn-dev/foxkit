package rename

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func mk(t *testing.T, files ...string) string {
	t.Helper()
	d := t.TempDir()
	for _, f := range files {
		p := filepath.Join(d, f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func names(t *testing.T, d string) string {
	t.Helper()
	des, _ := os.ReadDir(d)
	var n []string
	for _, e := range des {
		n = append(n, e.Name())
	}
	sort.Strings(n)
	return strings.Join(n, ",")
}

func TestFindReplaceAndRegex(t *testing.T) {
	d := mk(t, "IMG_001.JPG", "IMG_002.JPG", "other.txt")
	p, err := Build(Options{Dir: d, Find: "IMG_", Replace: "trip_"})
	if err != nil || len(p.Changes) != 2 || p.Unchanged != 1 || len(p.Conflicts) != 0 {
		t.Fatalf("%+v %v", p, err)
	}
	if n, err := Apply(p); err != nil || n != 2 || names(t, d) != "other.txt,trip_001.JPG,trip_002.JPG" {
		t.Fatalf("n=%d err=%v %s", n, err, names(t, d))
	}
	d = mk(t, "a12.txt")
	p, _ = Build(Options{Dir: d, Find: `(\d+)`, Replace: "photo_$1", Regex: true})
	if p.Changes[0].To != "photo_a12.txt" && p.Changes[0].To != "aphoto_12.txt" {
		t.Fatalf("%+v", p.Changes)
	}
	if p.Changes[0].To != "aphoto_12.txt" {
		t.Fatalf("$1 phải được thay: %s", p.Changes[0].To)
	}
	p, _ = Build(Options{Dir: d, Find: "A", Replace: "b", IgnoreCase: true})
	if p.Changes[0].To != "b12.txt" {
		t.Fatalf("ignore-case: %s", p.Changes[0].To)
	}
	if _, err := Build(Options{Dir: d, Find: "(", Replace: "x", Regex: true}); err == nil {
		t.Fatal("regex lỗi phải báo")
	}
}

func TestSwapAndCaseOnlyRename(t *testing.T) {
	d := mk(t, "a.txt", "b.txt")
	p := Plan{Changes: []Change{{d, "a.txt", "b.txt"}, {d, "b.txt", "a.txt"}}, Conflicts: []Conflict{}}
	if _, err := Apply(p); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(filepath.Join(d, "a.txt")); string(got) != "b.txt" {
		t.Fatalf("hoán đổi sai: a.txt chứa %q", got)
	}
	d = mk(t, "a.txt")
	p, _ = Build(Options{Dir: d, Case: "upper"})
	if len(p.Changes) != 1 || p.Changes[0].To != "A.txt" || len(p.Conflicts) != 0 {
		t.Fatalf("a.txt→A.txt (khác hoa thường, không phải xung đột): %+v", p)
	}
	if _, err := Apply(p); err != nil || names(t, d) != "A.txt" {
		t.Fatalf("%v %s", err, names(t, d))
	}
}

func TestConflicts(t *testing.T) {
	d := mk(t, "a1.txt", "a2.txt")
	p, _ := Build(Options{Dir: d, Find: "1", Replace: "X"})
	p2, _ := Build(Options{Dir: d, Find: "[12]", Replace: "X", Regex: true})
	if len(p.Conflicts) != 0 || len(p2.Conflicts) != 1 || p2.Conflicts[0].Reason != "trùng tên đích" {
		t.Fatalf("%+v / %+v", p, p2)
	}
	if _, err := Apply(p2); !errors.Is(err, ErrConflict) {
		t.Fatalf("Apply phải từ chối: %v", err)
	}
	if names(t, d) != "a1.txt,a2.txt" {
		t.Fatal("không được đổi gì khi xung đột")
	}
	// đích đã tồn tại và không nằm trong danh sách đổi
	d = mk(t, "a.txt", "b.txt")
	p, _ = Build(Options{Dir: d, Find: "a", Replace: "b"})
	if len(p.Conflicts) != 1 || p.Conflicts[0].Reason != "file đã tồn tại" {
		t.Fatalf("%+v", p.Conflicts)
	}
	// tên không hợp lệ
	d = mk(t, "x.txt", "y.txt")
	for _, to := range []string{"CON", "a?"} {
		p, _ = Build(Options{Dir: d, Find: "x", Replace: to})
		if len(p.Conflicts) != 1 || p.Conflicts[0].Reason != "tên không hợp lệ" {
			t.Fatalf("%s: %+v", to, p.Conflicts)
		}
	}
}

func TestNumberingNaturalOrder(t *testing.T) {
	d := mk(t, "img2.jpg", "img10.jpg", "img1.jpg")
	p, err := Build(Options{Dir: d, Number: "Trip_{n:3}", Sort: "name", Start: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(p); err != nil {
		t.Fatal(err)
	}
	for want, orig := range map[string]string{"Trip_001.jpg": "img1.jpg", "Trip_002.jpg": "img2.jpg", "Trip_003.jpg": "img10.jpg"} {
		if b, _ := os.ReadFile(filepath.Join(d, want)); string(b) != orig {
			t.Errorf("%s chứa %q, cần %q", want, b, orig)
		}
	}
}

func TestRollbackOnPhase2Failure(t *testing.T) {
	d := mk(t, "a.txt", "b.txt", "c.txt")
	p, _ := Build(Options{Dir: d, Find: ".txt", Replace: ".md", WithExt: true})
	calls := 0
	renameFn = func(a, b string) error {
		calls++
		if calls == 5 { // 3 lần pha 1, lần 4 pha 2 ok, lần 5 lỗi
			return errors.New("giả lập: file đang mở")
		}
		return os.Rename(a, b)
	}
	defer func() { renameFn = os.Rename }()
	if _, err := Apply(p); err == nil {
		t.Fatal("cần lỗi")
	}
	if names(t, d) != "a.txt,b.txt,c.txt" {
		t.Fatalf("phải hoàn tác về tên gốc, đang là %s", names(t, d))
	}
}

func TestDirsRenamedLastDeepestFirst(t *testing.T) {
	d := mk(t, "old/sub/f.txt", "old/g.txt")
	p, err := Build(Options{Dir: d, Find: "old", Replace: "new", Recursive: true, Dirs: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(d, "new", "sub", "f.txt")); err != nil {
		t.Fatalf("%v; cây hiện tại: %s", err, names(t, d))
	}
}

func TestValidNameAndPattern(t *testing.T) {
	for n, want := range map[string]bool{"ok.txt": true, "CON.txt": false, "com1": false, "a.": false, "a ": false, "": false, "a<b": false, "con1.txt": true} {
		if ValidName(n) != want {
			t.Errorf("ValidName(%q) != %v", n, want)
		}
	}
	if !ValidatePattern("x_{n}") || !ValidatePattern("x_{n:3}") || ValidatePattern("x") {
		t.Error("ValidatePattern sai")
	}
}
