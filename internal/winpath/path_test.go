package winpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyze(t *testing.T) {
	root := t.TempDir()
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "x", "b")
	for _, d := range []string{a, b} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("TMPX", filepath.Join(root, "x"))
	in := a + ";;" + a + `\;%TMPX%\b;` + filepath.Join(root, "missing")
	r := Analyze(in)
	if len(r.Entries) != 5 || r.Length != len(in) {
		t.Fatalf("%+v", r)
	}
	if !r.Entries[1].Empty {
		t.Error("mục 2 phải rỗng")
	}
	if !r.Entries[2].Duplicate || r.Entries[0].Duplicate {
		t.Error("mục 3 trùng mục 1 (khác hoa thường/dấu \\ cuối)")
	}
	if r.Entries[3].Expanded != b || !r.Entries[3].Exists {
		t.Errorf("expand sai: %+v", r.Entries[3])
	}
	if r.Entries[4].Exists {
		t.Error("mục cuối không tồn tại")
	}
	if r.Problems != 3 || len(r.Problematic()) != 3 {
		t.Errorf("problems = %d", r.Problems)
	}
}

func TestEmptyPathAndExpand(t *testing.T) {
	r := Analyze("")
	if r.Entries == nil || len(r.Entries) != 0 || r.Problems != 0 {
		t.Fatalf("%+v", r)
	}
	if got := Expand("%NOPE_UNSET_XYZ%\\a"); got != "%NOPE_UNSET_XYZ%\\a" {
		t.Errorf("biến lạ giữ nguyên, got %q", got)
	}
}
