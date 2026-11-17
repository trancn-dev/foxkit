package size

import (
	"context"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func write(t *testing.T, p string, n int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(strings.Repeat("x", n)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func tree(t *testing.T) string {
	d := t.TempDir()
	for i := 0; i < 3; i++ {
		write(t, filepath.Join(d, "a", string(rune('1'+i))+".txt"), 100)
	}
	write(t, filepath.Join(d, "b", "c", "big.bin"), 1000)
	write(t, filepath.Join(d, "x.bin"), 50)
	return d
}

func TestByChildren(t *testing.T) {
	d := tree(t)
	r, err := ByChildren(context.Background(), d, 20)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, it := range r.Items {
		names = append(names, it.Name)
	}
	if strings.Join(names, ",") != "b,a,x.bin" || r.Total != 1350 {
		t.Fatalf("names %v total %d", names, r.Total)
	}
	if r.Items[0].Pct != 74.1 {
		t.Fatalf("pct = %v", r.Items[0].Pct)
	}
	r, _ = ByChildren(context.Background(), d, 1)
	if len(r.Items) != 2 || r.Items[1].Name != "(còn lại 2 mục)" || r.Items[1].Bytes != 350 {
		t.Fatalf("gộp phần còn lại sai: %+v", r.Items)
	}
}

func TestLargestFiles(t *testing.T) {
	r, err := LargestFiles(context.Background(), tree(t), 2)
	if err != nil || len(r.Items) != 2 || r.Items[0].Bytes != 1000 || r.Items[1].Bytes != 100 {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestHeapMatchesFullSort(t *testing.T) {
	d := t.TempDir()
	rng := rand.New(rand.NewSource(1))
	var sizes []int
	for i := 0; i < 60; i++ {
		n := rng.Intn(5000) + 1
		sizes = append(sizes, n)
		write(t, filepath.Join(d, "d", string(rune('a'+i%5)), "f"+string(rune('A'+i%26))+string(rune('a'+i/26))+".bin"), n)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sizes)))
	r, err := LargestFiles(context.Background(), d, 10)
	if err != nil {
		t.Fatal(err)
	}
	for i, it := range r.Items {
		if int(it.Bytes) != sizes[i] {
			t.Fatalf("vị trí %d: heap %d, sort %d", i, it.Bytes, sizes[i])
		}
	}
}

func TestCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ByChildren(ctx, tree(t), 5); err == nil {
		t.Fatal("cần lỗi khi ctx đã hủy")
	}
}
