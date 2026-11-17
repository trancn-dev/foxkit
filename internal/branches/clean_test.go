package branches

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func commit(t *testing.T, d, file string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(d, file), []byte(file), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, d, "add", ".")
	gitIn(t, d, "commit", "-q", "-m", file)
}

func repo(t *testing.T) string {
	if !git.Available() {
		t.Skip("không có git")
	}
	d := t.TempDir()
	gitIn(t, d, "init", "-q", "-b", "main")
	commit(t, d, "init.txt")
	gitIn(t, d, "branch", "develop")
	gitIn(t, d, "checkout", "-q", "-b", "feature/a")
	commit(t, d, "a.txt")
	gitIn(t, d, "checkout", "-q", "main")
	gitIn(t, d, "merge", "-q", "--no-ff", "-m", "merge a", "feature/a")
	gitIn(t, d, "checkout", "-q", "-b", "feature/b")
	commit(t, d, "b.txt")
	gitIn(t, d, "branch", "release/1.0")
	gitIn(t, d, "branch", "keepme", "main")
	return d
}

func names(cs []Candidate) string {
	var n []string
	for _, c := range cs {
		n = append(n, c.Name)
	}
	return strings.Join(n, ",")
}

func TestBuildAndApply(t *testing.T) {
	d := repo(t)
	p, err := Build(context.Background(), d, "", false, []string{"keepme"})
	if err != nil {
		t.Fatal(err)
	}
	if p.Base != "main" || names(p.Delete) != "feature/a" || p.Delete[0].Reason != "merged" || p.Delete[0].Last == "" {
		t.Fatalf("%+v", p)
	}
	kept := strings.Join(p.Kept, ",")
	for _, want := range []string{"develop", "main", "feature/b", "keepme", "release/1.0"} {
		if !strings.Contains(kept, want) {
			t.Errorf("%s phải được giữ, kept = %s", want, kept)
		}
	}
	if err := Apply(context.Background(), d, &p); err != nil || !p.Applied || len(p.Failed) != 0 {
		t.Fatalf("%+v %v", p, err)
	}
	out, _ := exec.Command("git", "-C", d, "branch", "--format=%(refname:short)").Output()
	if strings.Contains(string(out), "feature/a") || !strings.Contains(string(out), "feature/b") {
		t.Fatalf("sau apply:\n%s", out)
	}
}

func TestCurrentAndUnmergedNeverDeleted(t *testing.T) {
	d := repo(t)
	p, _ := Build(context.Background(), d, "", false, nil)
	for _, c := range p.Delete {
		if c.Name == "feature/b" || c.Name == "main" {
			t.Fatalf("không được xóa %s", c.Name)
		}
	}
}

func TestErrors(t *testing.T) {
	if !git.Available() {
		t.Skip("không có git")
	}
	if _, err := Build(context.Background(), t.TempDir(), "", false, nil); !errors.Is(err, ErrNotRepo) {
		t.Errorf("không phải repo: %v", err)
	}
	d := repo(t)
	if _, err := Build(context.Background(), d, "nope-branch", false, nil); !errors.Is(err, ErrBaseNotFound) {
		t.Errorf("base lạ: %v", err)
	}
}
