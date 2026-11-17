package zipx

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func put(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// tree trả map đường dẫn tương đối → hash nội dung (thư mục rỗng có giá trị "dir").
func tree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		if info.IsDir() {
			out[filepath.ToSlash(rel)] = "dir"
			return nil
		}
		b, _ := os.ReadFile(p)
		h := sha256.Sum256(b)
		out[filepath.ToSlash(rel)] = string(h[:4])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func makeZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "evil.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		w, err := zw.Create(n)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(entries[n]))
	}
	zw.Close()
	f.Close()
	return p
}

func TestRoundTripVietnameseAndEmptyDir(t *testing.T) {
	root := t.TempDir()
	shop := filepath.Join(root, "shop")
	put(t, shop, "Đơn hàng.txt", "xin chào")
	put(t, shop, "src/app/main.js", strings.Repeat("a", 5000))
	put(t, shop, "assets/logo.png", "PNGDATA")
	if err := os.MkdirAll(filepath.Join(shop, "thư mục rỗng"), 0o755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "shop.zip")
	res, err := Pack(context.Background(), []string{shop}, out, PackOptions{Level: 6})
	if err != nil || res.Files != 3 || res.ZipBytes == 0 {
		t.Fatalf("%+v %v", res, err)
	}
	entries, err := List(out)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
		if strings.Contains(e.Name, `\`) {
			t.Errorf("tên trong zip phải dùng /: %q", e.Name)
		}
	}
	joined := strings.Join(names, "|")
	for _, want := range []string{"shop/Đơn hàng.txt", "shop/thư mục rỗng/", "shop/src/app/main.js"} {
		if !strings.Contains(joined, want) {
			t.Errorf("thiếu %q trong %v", want, names)
		}
	}
	to := filepath.Join(root, "out")
	ur, err := Unpack(context.Background(), out, to, false)
	if err != nil || ur.Files != 3 {
		t.Fatalf("%+v %v", ur, err)
	}
	want, got := tree(t, shop), tree(t, filepath.Join(to, "shop"))
	if len(want) != len(got) {
		t.Fatalf("cây khác nhau:\nwant %v\ngot  %v", want, got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s khác", k)
		}
	}
}

func TestDefaultExcludesAndOverride(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "proj")
	put(t, p, "index.php", "<?php")
	put(t, p, ".env", "SECRET=1")
	put(t, p, "debug.log", "x")
	put(t, p, "a/b.log", "y")
	put(t, p, "node_modules/x/y.js", "z")
	put(t, p, ".git/HEAD", "ref")
	put(t, p, "storage/framework/cache/c.dat", "c")
	res, err := Pack(context.Background(), []string{p}, filepath.Join(root, "p.zip"), PackOptions{Level: 6, Exclude: []string{"storage/framework/cache"}})
	if err != nil || res.Files != 1 {
		t.Fatalf("chỉ index.php được nén: %+v %v", res, err)
	}
	ex := strings.Join(res.Excluded, ",")
	for _, want := range []string{".env", ".git", "node_modules", "*.log ×2", "storage/framework/cache"} {
		if !strings.Contains(ex, want) {
			t.Errorf("Excluded thiếu %q: %s", want, ex)
		}
	}
	res, err = Pack(context.Background(), []string{p}, filepath.Join(root, "all.zip"), PackOptions{Level: 6, NoDefaultExcludes: true})
	if err != nil || res.Files != 7 || len(res.Excluded) != 0 {
		t.Fatalf("--no-default-excludes: %+v %v", res, err)
	}
}

func TestPackDoesNotIncludeItself(t *testing.T) {
	p := t.TempDir()
	put(t, p, "a.txt", "a")
	res, err := Pack(context.Background(), []string{p}, filepath.Join(p, "self.zip"), PackOptions{Level: 6})
	if err != nil || res.Files != 1 {
		t.Fatalf("out nằm trong nguồn không được tự nén chính nó: %+v %v", res, err)
	}
}

func TestZipSlipIsRejectedBeforeWritingAnything(t *testing.T) {
	for _, name := range []string{"../../evil.txt", `C:\evil.txt`, "/abs/evil.txt", "a/../../evil.txt", `..\..\evil.txt`, "CON.txt", "a:b.txt"} {
		z := makeZip(t, map[string]string{"safe.txt": "ok", name: "evil"})
		to := filepath.Join(t.TempDir(), "dest")
		_, err := Unpack(context.Background(), z, to, false)
		if !errors.Is(err, ErrUnsafePath) {
			t.Errorf("%q phải bị chặn, got %v", name, err)
		}
		if _, statErr := os.Stat(filepath.Join(to, "safe.txt")); statErr == nil {
			t.Errorf("%q: không được ghi file nào khi có entry nguy hiểm", name)
		}
	}
}

func TestZipBombRejected(t *testing.T) {
	z := makeZip(t, map[string]string{"zeros.bin": strings.Repeat("\x00", 20<<20)})
	_, err := Unpack(context.Background(), z, filepath.Join(t.TempDir(), "d"), false)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("tỉ lệ nén cực cao phải bị chặn: %v", err)
	}
}

func TestExistingFilesNeedForce(t *testing.T) {
	z := makeZip(t, map[string]string{"a.txt": "mới", "b.txt": "b"})
	to := t.TempDir()
	put(t, to, "a.txt", "cũ")
	_, err := Unpack(context.Background(), z, to, false)
	if !errors.Is(err, ErrExists) || !strings.Contains(err.Error(), "a.txt") {
		t.Fatalf("%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(to, "b.txt")); statErr == nil {
		t.Error("không được ghi b.txt trước khi báo file đã tồn tại")
	}
	if _, err := Unpack(context.Background(), z, to, true); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(to, "a.txt")); string(b) != "mới" {
		t.Errorf("--force phải ghi đè: %q", b)
	}
}

func TestPackFailureLeavesNoTmpOrZip(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(root, "x.zip")
	if _, err := Pack(context.Background(), []string{filepath.Join(root, "khong-co")}, out, PackOptions{Level: 6}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	src := filepath.Join(root, "s")
	put(t, src, "a.txt", "a")
	if _, err := Pack(ctx, []string{src}, out, PackOptions{Level: 6}); err == nil {
		t.Fatal("ctx hủy phải lỗi")
	}
	if entries, _ := os.ReadDir(root); len(entries) != 1 { // chỉ còn thư mục s
		t.Errorf("không được để lại .tmp/zip dở: %v", entries)
	}
	put(t, root, "exists.zip", "x")
	if _, err := Pack(context.Background(), []string{src}, filepath.Join(root, "exists.zip"), PackOptions{Level: 6}); !errors.Is(err, ErrExists) {
		t.Errorf("zip đích đã có không --force: %v", err)
	}
}

func TestInvalidZip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.zip")
	os.WriteFile(p, bytes.Repeat([]byte("x"), 100), 0o644)
	if _, err := List(p); !errors.Is(err, ErrInvalid) {
		t.Fatalf("%v", err)
	}
	if _, err := Unpack(context.Background(), p, t.TempDir(), false); !errors.Is(err, ErrInvalid) {
		t.Fatalf("%v", err)
	}
}
