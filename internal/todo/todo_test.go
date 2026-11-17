package todo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	re := Matcher(DefaultTags)
	tests := []struct {
		line, tag, text string
		ok              bool
	}{
		{"// TODO: validate coupon", "TODO", "validate coupon", true},
		{"# FIXME(tran): lỗi múi giờ", "FIXME", "lỗi múi giờ", true},
		{"/* HACK tạm thời */", "HACK", "tạm thời", true},
		{"{{-- TODO đổi icon --}}", "TODO", "đổi icon", true},
		{"<!-- XXX xem lại -->", "XXX", "xem lại", true},
		{" * TODO in docblock", "TODO", "in docblock", true},
		{"$todoList = [];", "", "", false},
		{"echo 'TODO';", "", "", false},
		{"// todo viết thường", "", "", false},
	}
	for _, tc := range tests {
		tag, text, ok := ParseLine(re, tc.line)
		if ok != tc.ok || tag != tc.tag || text != tc.text {
			t.Errorf("%q → (%q,%q,%v), cần (%q,%q,%v)", tc.line, tag, text, ok, tc.tag, tc.text, tc.ok)
		}
	}
	if _, text, _ := ParseLine(re, "// TODO "+strings.Repeat("a", 300)); len([]rune(text)) != 200 {
		t.Errorf("phải cắt 200 ký tự, got %d", len([]rune(text)))
	}
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunWalkFilterAndLongLine(t *testing.T) {
	d := t.TempDir()
	write(t, d, "app/A.php", "<?php\n// TODO: một\n// FIXME hai\n")
	write(t, d, "resources/Cart.vue", "<!-- HACK ba -->\n")
	write(t, d, "node_modules/x.js", "// TODO bỏ qua\n")
	write(t, d, "public/build/app.js", "// TODO bỏ qua\n")
	write(t, d, "a.min.js", "// TODO bỏ qua\n")
	write(t, d, "notes.txt", "// TODO sai đuôi\n")
	write(t, d, "views/x.blade.php", "{{-- TODO blade --}}\n")
	write(t, d, "long.js", "// "+strings.Repeat("x", 2<<20-100)+"\n// TODO sau dòng dài\n")
	r, err := Run(context.Background(), Options{Dir: d})
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, it := range r.Items {
		got = append(got, it.File+":"+it.Tag)
	}
	want := "app/A.php:TODO,app/A.php:FIXME,resources/Cart.vue:HACK,views/x.blade.php:TODO"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %v", got)
	}
	if r.Counts["TODO"] != 2 || r.Counts["FIXME"] != 1 || r.Items[0].Line != 2 {
		t.Fatalf("%+v", r)
	}
	r, _ = Run(context.Background(), Options{Dir: d, Tags: []string{"HACK"}})
	if len(r.Items) != 1 || len(r.Counts) != 1 {
		t.Fatalf("--tags: %+v", r)
	}
}

func TestValidTags(t *testing.T) {
	if ValidTags(nil) || ValidTags([]string{""}) || ValidTags([]string{"A B"}) || !ValidTags([]string{"TODO", "NB_1"}) {
		t.Error("ValidTags sai")
	}
}
