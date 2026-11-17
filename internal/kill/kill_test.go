package kill

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"foxkit/internal/ps"
	"foxkit/internal/winapi"
)

func fake(procs ...ps.Proc) {
	listFn = func(context.Context) ([]ps.Proc, error) { return procs, nil }
}

func codes(f []Failure) string {
	var c []string
	for _, x := range f {
		c = append(c, x.Code)
	}
	return strings.Join(c, ",")
}

func TestResolve(t *testing.T) {
	fake(
		ps.Proc{PID: 100, Name: "node.exe"}, ps.Proc{PID: 101, Name: "node.exe"}, ps.Proc{PID: 200, Name: "notepad.exe"},
		ps.Proc{PID: 300, Name: "svchost.exe"}, ps.Proc{PID: 4, Name: "System"}, ps.Proc{PID: 400, Name: "explorer.exe"},
	)
	tests := []struct {
		name       string
		args       []string
		wantTarget int
		wantCodes  string
	}{
		{"name gets .exe, 2 PIDs", []string{"node"}, 2, ""},
		{"case-insensitive", []string{"NODE.EXE"}, 2, ""},
		{"exact not substring", []string{"note"}, 0, "PROC_NOT_FOUND"},
		{"pid", []string{"200"}, 1, ""},
		{"unknown pid", []string{"999"}, 0, "PROC_NOT_FOUND"},
		{"protected svchost", []string{"svchost"}, 0, "PROC_PROTECTED"},
		{"protected explorer", []string{"explorer"}, 0, "PROC_PROTECTED"},
		{"protected System", []string{"system"}, 0, "PROC_PROTECTED"},
		{"pid 4", []string{"4"}, 0, "PROC_PROTECTED"},
		{"mixed keeps going", []string{"abc", "node"}, 2, "PROC_NOT_FOUND"},
		{"dedupe", []string{"100", "node"}, 2, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts, f, err := Resolve(context.Background(), tc.args)
			if err != nil || len(ts) != tc.wantTarget || codes(f) != tc.wantCodes {
				t.Fatalf("targets %v failed %v err %v", ts, f, err)
			}
		})
	}
}

func TestSelfAndParentProtected(t *testing.T) {
	fake(ps.Proc{PID: os.Getpid(), Name: "foxkit.exe"}, ps.Proc{PID: os.Getppid(), Name: "pwsh.exe"})
	ts, f, _ := Resolve(context.Background(), []string{"foxkit", "pwsh"})
	if len(ts) != 0 || codes(f) != "PROC_PROTECTED,PROC_PROTECTED" {
		t.Fatalf("%v %v", ts, f)
	}
}

func TestKillResultAndExitCode(t *testing.T) {
	old := killFn
	defer func() { killFn = old }()
	killFn = func(pid int) error {
		switch pid {
		case 1:
			return nil
		case 2:
			return os.ErrPermission
		}
		return errors.New("hỏng")
	}
	r := Kill(context.Background(), []Target{{1, "a.exe"}, {2, "b.exe"}}, false)
	if len(r.Killed) != 1 || len(r.Failed) != 1 || r.Failed[0].Code != "NEED_ADMIN" || ExitCode(r) != "NEED_ADMIN" {
		t.Fatalf("%+v", r)
	}
	r = Kill(context.Background(), []Target{{2, "b.exe"}, {3, "c.exe"}}, false)
	if ExitCode(r) != "KILL_PARTIAL" {
		t.Fatalf("%+v", r)
	}
	if ExitCode(Kill(context.Background(), []Target{{1, "a.exe"}}, false)) != "" {
		t.Fatal("thành công phải không có mã lỗi")
	}
	if got := Kill(context.Background(), []Target{{4, "System"}}, false); got.Failed[0].Code != "PROC_PROTECTED" {
		t.Fatalf("%+v", got)
	}
}

func TestTreeKillsChildrenBeforeParent(t *testing.T) {
	oldK, oldP := killFn, procsFn
	defer func() { killFn, procsFn = oldK, oldP }()
	// npm(10) → node(11) → esbuild(12); process 20 không liên quan
	procsFn = func() ([]winapi.Proc, error) {
		return []winapi.Proc{{PID: 10, PPID: 1}, {PID: 11, PPID: 10}, {PID: 12, PPID: 11}, {PID: 20, PPID: 1}}, nil
	}
	var order []int
	killFn = func(pid int) error { order = append(order, pid); return nil }
	r := Kill(context.Background(), []Target{{10, "npm.exe"}}, true)
	if len(r.Killed) != 1 || len(order) != 3 || order[0] != 12 || order[1] != 11 || order[2] != 10 {
		t.Fatalf("phải kill cháu → con → cha, got %v %+v", order, r)
	}
}
