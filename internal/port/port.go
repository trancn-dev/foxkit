package port

import (
	"context"
	"errors"
	"os"
	"time"

	"foxkit/internal/winexec"
)

type Listener struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	PID      int    `json:"pid"`
	Process  string `json:"process,omitempty"`
	Path     string `json:"path,omitempty"` // GĐ6 (winapi) sẽ điền
}

type Result struct {
	Port      int        `json:"port"`
	InUse     bool       `json:"inUse"`
	Listeners []Listener `json:"listeners"`
	Killed    []int      `json:"killed,omitempty"`
}

var ErrSystemProcess = errors.New("process hệ thống (PID 0/4)")

// List liệt kê mọi socket TCP đang LISTEN, kèm tên process (một lần gọi tasklist).
func List(ctx context.Context) ([]Listener, error) {
	out, err := winexec.Run(ctx, 5*time.Second, "netstat", "-ano")
	if err != nil {
		return nil, err
	}
	items := ParseNetstat(string(out))
	if len(items) == 0 {
		return items, nil
	}
	names := map[int]string{}
	if procs, err := winexec.Tasklist(ctx); err == nil { // không tra được tên thì để trống, không coi là lỗi
		for _, p := range procs {
			names[p.PID] = p.Name
		}
	}
	for i := range items {
		items[i].Process = names[items[i].PID]
	}
	return items, nil
}

// Find trả các listener của một port.
func Find(ctx context.Context, port int) (Result, error) {
	all, err := List(ctx)
	if err != nil {
		return Result{}, err
	}
	res := Result{Port: port, Listeners: make([]Listener, 0)}
	for _, l := range all {
		if l.Port == port {
			res.Listeners = append(res.Listeners, l)
		}
	}
	res.InUse = len(res.Listeners) > 0
	return res, nil
}

// Kill kết thúc process theo PID. PID 0/4 bị chặn bằng ErrSystemProcess.
func Kill(pid int) error {
	if pid == 0 || pid == 4 {
		return ErrSystemProcess
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

// WaitFree chờ tối đa timeout tới khi port không còn listener.
func WaitFree(ctx context.Context, port int, timeout time.Duration) (Result, error) {
	deadline := time.Now().Add(timeout)
	for {
		res, err := Find(ctx, port)
		if err != nil || !res.InUse || time.Now().After(deadline) {
			return res, err
		}
		time.Sleep(200 * time.Millisecond)
	}
}
