package winapi

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestDescendants(t *testing.T) {
	procs := []Proc{{1, 0, "root"}, {2, 1, "a"}, {3, 2, "b"}, {4, 1, "c"}, {5, 9, "other"}, {0, 0, "idle"}}
	got := Descendants(procs, 1)
	if len(got) != 3 {
		t.Fatalf("cần 3 con cháu của 1, got %v", got)
	}
	if len(Descendants(procs, 5)) != 0 {
		t.Error("process không có con phải trả rỗng")
	}
}

func TestRealProcessAPIs(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ Windows")
	}
	p, err := ProcessPath(os.Getpid())
	if err != nil || !strings.HasSuffix(strings.ToLower(p), ".exe") {
		t.Fatalf("ProcessPath(self) = %q, %v", p, err)
	}
	procs, err := Processes()
	if err != nil || len(procs) == 0 {
		t.Fatalf("Processes: %v %v", len(procs), err)
	}
	found := false
	for _, pr := range procs {
		if pr.PID == os.Getpid() {
			found = pr.PPID == os.Getppid()
		}
	}
	if !found {
		t.Error("process hiện tại phải có trong snapshot với đúng PPID")
	}
	_ = IsAdmin() // chỉ kiểm không panic: giá trị phụ thuộc cách chạy
}
