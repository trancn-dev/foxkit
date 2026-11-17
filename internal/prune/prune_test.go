package prune

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const day = 24 * time.Hour

func put(t *testing.T, root, rel string, age time.Duration) string {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	mt := time.Now().Add(-age)
	if err := os.Chtimes(p, mt, mt); err != nil {
		t.Fatal(err)
	}
	return p
}

func opts(d string) Options {
	return Options{Dir: d, Targets: []string{"node_modules", "vendor"}, Older: 30 * day, Depth: 4}
}

func fixture(t *testing.T) string {
	d := t.TempDir()
	put(t, d, "p1/package.json", 60*day)
	put(t, d, "p1/node_modules/x/y.js", 0)
	put(t, d, "p2/package.json", day)
	put(t, d, "p2/node_modules/a.js", 0)
	put(t, d, "go-proj/go.mod", 90*day)
	put(t, d, "go-proj/vendor/m/m.go", 90*day)
	put(t, d, "p3/package.json", 60*day)
	put(t, d, "p3/node_modules/a/node_modules/b.js", 0)
	return d
}

func TestScanFindsOnlyOldProjects(t *testing.T) {
	d := fixture(t)
	r, err := Run(context.Background(), opts(d))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Items) != 2 || r.Skipped != 1 {
		t.Fatalf("%+v", r)
	}
	for _, it := range r.Items {
		base := filepath.Base(it.Project)
		if base != "p1" && base != "p3" {
			t.Errorf("không được chọn %s", it.Path)
		}
		if it.Deleted || it.Bytes != 1 {
			t.Errorf("%+v", it)
		}
	}
	if r.TotalBytes != 2 || r.Items == nil {
		t.Fatalf("%+v", r)
	}
	if _, err := os.Stat(filepath.Join(d, "go-proj", "vendor")); err != nil {
		t.Fatal("vendor của Go không được đụng")
	}
}

func TestOlderZeroMeansAllProjects(t *testing.T) {
	o := opts(fixture(t))
	o.Older = 0
	r, _ := Run(context.Background(), o)
	if len(r.Items) != 3 || r.Skipped != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestNestedTargetOnlyOuterLevel(t *testing.T) {
	r, _ := Run(context.Background(), opts(fixture(t)))
	for _, it := range r.Items {
		if filepath.Base(it.Project) == "p3" && filepath.Base(filepath.Dir(it.Path)) != "p3" {
			t.Fatalf("ghi nhận target lồng: %s", it.Path)
		}
	}
}

func TestApplyDeletesIncludingReadOnly(t *testing.T) {
	d := fixture(t)
	ro := put(t, d, "p1/node_modules/ro/file.txt", 0)
	if err := os.Chmod(ro, 0o444); err != nil {
		t.Fatal(err)
	}
	o := opts(d)
	o.Apply = true
	r, err := Run(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range r.Items {
		if !it.Deleted || it.Error != "" {
			t.Fatalf("%+v", it)
		}
		if _, err := os.Stat(it.Path); err == nil {
			t.Fatalf("%s vẫn còn", it.Path)
		}
	}
	if _, err := os.Stat(filepath.Join(d, "p2", "node_modules")); err != nil {
		t.Fatal("dự án mới không được xóa")
	}
	if r.FreedBytes != r.TotalBytes {
		t.Fatalf("%+v", r)
	}
}

func TestDepthLimit(t *testing.T) {
	d := t.TempDir()
	put(t, d, "a/b/c/d/package.json", 60*day)
	put(t, d, "a/b/c/d/node_modules/x.js", 0)
	o := opts(d)
	o.Depth = 2
	if r, _ := Run(context.Background(), o); len(r.Items) != 0 {
		t.Fatalf("depth 2 không tới được: %+v", r)
	}
	o.Depth = 5
	if r, _ := Run(context.Background(), o); len(r.Items) != 1 {
		t.Fatalf("depth 5 phải thấy: %+v", r)
	}
}
