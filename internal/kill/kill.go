package kill

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"foxkit/internal/ps"
	"foxkit/internal/winapi"
)

type Target struct {
	PID  int    `json:"pid"`
	Name string `json:"name"`
}

type Failure struct {
	Target
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	Killed []Target  `json:"killed"`
	Failed []Failure `json:"failed"`
}

// listFn tách ra để test không cần tasklist thật.
var listFn = ps.List

// killFn / treeFn: thực thi thật, cũng tách ra để test.
var killFn = func(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

// procsFn: snapshot process (có PPID) để tự duyệt cây; tách ra để test.
var procsFn = winapi.Processes

// treeFn kill process con/cháu (sâu nhất trước) rồi tới chính nó, thay cho `taskkill /T`. Lỗi đầu tiên được trả sau khi đã thử hết.
var treeFn = func(ctx context.Context, pid int) error {
	procs, err := procsFn()
	if err != nil {
		return err
	}
	order := winapi.Descendants(procs, pid) // BFS: cha trước con → kill ngược lại để con trước
	var first error
	for i := len(order) - 1; i >= 0; i-- {
		if err := killFn(order[i]); err != nil && first == nil && !errors.Is(err, os.ErrProcessDone) {
			first = err
		}
	}
	if err := killFn(pid); err != nil && first == nil {
		first = err
	}
	return first
}

var protectedNames = map[string]bool{
	"system": true, "smss.exe": true, "csrss.exe": true, "wininit.exe": true, "winlogon.exe": true,
	"services.exe": true, "lsass.exe": true, "svchost.exe": true, "explorer.exe": true, "dwm.exe": true,
}

func protected(t Target) bool {
	return t.PID == 0 || t.PID == 4 || t.PID == os.Getpid() || t.PID == os.Getppid() || protectedNames[strings.ToLower(t.Name)]
}

// Resolve đổi tham số (PID hoặc tên) thành Target qua một lần gọi ps.List.
// Tên thiếu .exe tự thêm; so khớp chính xác không phân biệt hoa thường. Mục không tìm thấy/bị chặn nằm trong failed.
func Resolve(ctx context.Context, args []string) (targets []Target, failed []Failure, err error) {
	procs, err := listFn(ctx)
	if err != nil {
		return nil, nil, err
	}
	byName := map[string][]Target{}
	byPID := map[int]Target{}
	for _, p := range procs {
		t := Target{PID: p.PID, Name: p.Name}
		byName[strings.ToLower(p.Name)] = append(byName[strings.ToLower(p.Name)], t)
		byPID[p.PID] = t
	}
	seen := map[int]bool{}
	targets, failed = []Target{}, []Failure{}
	add := func(t Target) {
		if seen[t.PID] {
			return
		}
		seen[t.PID] = true
		if protected(t) {
			failed = append(failed, Failure{Target: t, Code: "PROC_PROTECTED", Message: "Process hệ thống, không được kill"})
			return
		}
		targets = append(targets, t)
	}
	for _, a := range args {
		if pid, err := strconv.Atoi(a); err == nil {
			t, ok := byPID[pid]
			if !ok {
				failed = append(failed, Failure{Target: Target{PID: pid, Name: a}, Code: "PROC_NOT_FOUND", Message: fmt.Sprintf("Không có process PID %d", pid)})
				continue
			}
			add(t)
			continue
		}
		name := strings.ToLower(a)
		if !strings.HasSuffix(name, ".exe") {
			name += ".exe"
		}
		ts := byName[name]
		if len(ts) == 0 && protectedNames[strings.ToLower(a)] { // "system" không có đuôi .exe
			ts = byName[strings.ToLower(a)]
		}
		if len(ts) == 0 {
			failed = append(failed, Failure{Target: Target{Name: a}, Code: "PROC_NOT_FOUND", Message: "Không có process tên " + a})
			continue
		}
		for _, t := range ts {
			add(t)
		}
	}
	return targets, failed, nil
}

// Kill kết thúc từng target; lỗi từng mục nằm trong Failed, không dừng các mục khác.
func Kill(ctx context.Context, targets []Target, tree bool) Result {
	res := Result{Killed: []Target{}, Failed: []Failure{}}
	for _, t := range targets {
		if protected(t) { // kiểm lần nữa phòng khi gọi trực tiếp không qua Resolve
			res.Failed = append(res.Failed, Failure{Target: t, Code: "PROC_PROTECTED", Message: "Process hệ thống, không được kill"})
			continue
		}
		var err error
		if tree {
			if err = treeFn(ctx, t.PID); err != nil {
				err = classify(killFn(t.PID), err) // taskkill lỗi: thử kill trơn để biết có phải do quyền không
			}
		} else {
			err = killFn(t.PID)
		}
		switch {
		case err == nil:
			res.Killed = append(res.Killed, t)
		case errors.Is(err, os.ErrPermission):
			res.Failed = append(res.Failed, Failure{Target: t, Code: "NEED_ADMIN", Message: "Cần quyền admin"})
		default:
			res.Failed = append(res.Failed, Failure{Target: t, Code: "KILL_FAILED", Message: err.Error()})
		}
	}
	return res
}

// classify: nếu kill trơn thành công (nil) coi như xong; nếu bị từ chối quyền trả lỗi quyền; còn lại giữ lỗi taskkill.
func classify(plain, taskkill error) error {
	if plain == nil || errors.Is(plain, os.ErrPermission) {
		return plain
	}
	return taskkill
}

// ExitError tổng hợp mã lỗi: nil nếu mọi thứ thành công; "NEED_ADMIN" nếu chỉ thất bại do quyền; ngược lại "KILL_PARTIAL".
func ExitCode(r Result) string {
	if len(r.Failed) == 0 {
		return ""
	}
	for _, f := range r.Failed {
		if f.Code != "NEED_ADMIN" {
			return "KILL_PARTIAL"
		}
	}
	return "NEED_ADMIN"
}
