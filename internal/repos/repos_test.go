package repos

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"foxkit/internal/git"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.email=t@t", "-c", "user.name=t", "-c", "commit.gpgsign=false"}, args...)
	if out, err := exec.Command("git", full...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func mkRepo(t *testing.T, root, name string, commit bool) string {
	d := filepath.Join(root, name)
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	gitIn(t, d, "init", "-q", "-b", "main")
	if commit {
		if err := os.WriteFile(filepath.Join(d, "a.txt"), []byte("a"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitIn(t, d, "add", ".")
		gitIn(t, d, "commit", "-q", "-m", "init")
	}
	return d
}

func TestScanFindsReposAndStatus(t *testing.T) {
	if !git.Available() {
		t.Skip("không có git")
	}
	root := t.TempDir()
	dirty := mkRepo(t, root, "dirty", true)
	if err := os.WriteFile(filepath.Join(dirty, "new.txt"), []byte("n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mkRepo(t, root, "clean", true)
	empty := mkRepo(t, root, "group/empty", false)
	// repo lồng không được quét
	mkRepo(t, empty, "nested", true)
	// node_modules bị bỏ qua
	mkRepo(t, root, "x/node_modules/pkg", true)

	rs, err := Scan(context.Background(), root, 3, false)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]Repo{}
	for _, r := range rs {
		names[r.Name] = r
	}
	if len(rs) != 3 || names["nested"].Path != "" || names["pkg"].Path != "" {
		t.Fatalf("phải đúng 3 repo (không lồng, không node_modules): %+v", rs)
	}
	if names["dirty"].Untracked != 1 || names["dirty"].Branch != "main" || names["dirty"].LastCommit == "" {
		t.Errorf("dirty: %+v", names["dirty"])
	}
	if names["empty"].LastCommit != "" || names["empty"].Error != "" {
		t.Errorf("repo chưa có commit: %+v", names["empty"])
	}
	if !names["clean"].Attention() { // sạch nhưng chưa có upstream → vẫn cần chú ý theo spec
		t.Error("không có upstream phải được tính là cần chú ý")
	}
	if rs[0].Attention() == false {
		t.Error("repo cần chú ý phải xếp trước")
	}
}

func TestDepthLimit(t *testing.T) {
	if !git.Available() {
		t.Skip("không có git")
	}
	root := t.TempDir()
	mkRepo(t, root, "a/b/c/deep", true)
	if rs, _ := Scan(context.Background(), root, 2, false); len(rs) != 0 {
		t.Fatalf("depth 2 không tới: %+v", rs)
	}
	if rs, _ := Scan(context.Background(), root, 4, false); len(rs) != 1 {
		t.Fatalf("depth 4 phải thấy: %+v", rs)
	}
}
