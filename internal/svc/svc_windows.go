package svc

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// connect chỉ xin quyền CONNECT + ENUMERATE (mgr.Connect() xin ALL_ACCESS nên lỗi khi không phải admin).
func connect() (*mgr.Mgr, error) {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT|windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		return nil, mapErr(err)
	}
	return &mgr.Mgr{Handle: h}, nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, windows.ERROR_ACCESS_DENIED):
		return ErrAccess
	case errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST):
		return ErrNotFound
	case errors.Is(err, windows.ERROR_DEPENDENT_SERVICES_RUNNING):
		return ErrDependents
	case errors.Is(err, windows.ERROR_SERVICE_DISABLED):
		return ErrDisabled
	}
	return err
}

// open mở một service với quyền tối thiểu cần dùng (least privilege).
func open(m *mgr.Mgr, name string, access uint32) (*mgr.Service, error) {
	p, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, err
	}
	h, err := windows.OpenService(m.Handle, p, access)
	if err != nil {
		return nil, mapErr(err)
	}
	return &mgr.Service{Name: name, Handle: h}, nil
}

func describe(s *mgr.Service) (Service, error) {
	st, err := s.Query()
	if err != nil {
		return Service{}, err
	}
	out := Service{Name: s.Name, State: StateName(uint32(st.State)), PID: st.ProcessId}
	if cfg, err := s.Config(); err == nil {
		out.Display, out.StartType = cfg.DisplayName, StartTypeName(cfg.StartType, cfg.DelayedAutoStart)
	}
	return out, nil
}

// List liệt kê service. all=false và filter rỗng → chỉ service dev; có filter → khớp name/display name.
func List(filter string, all bool) ([]Service, error) {
	m, err := connect()
	if err != nil {
		return nil, err
	}
	defer m.Disconnect()
	names, err := m.ListServices()
	if err != nil {
		return nil, mapErr(err)
	}
	out := []Service{}
	for _, n := range names {
		s, err := open(m, n, windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
		if err != nil {
			continue // service không mở được (quyền) thì bỏ qua
		}
		d, err := describe(s)
		s.Close()
		if err != nil {
			continue
		}
		switch {
		case filter != "" && !Matches(d.Name, d.Display, filter):
			continue
		case filter == "" && !all && !IsDev(d.Name, d.Display):
			continue
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}

// resolve: tìm service theo tên (không phân biệt hoa thường) hoặc display name khớp chính xác.
func resolve(m *mgr.Mgr, name string) (string, error) {
	names, err := m.ListServices()
	if err != nil {
		return "", mapErr(err)
	}
	for _, n := range names {
		if strings.EqualFold(n, name) {
			return n, nil
		}
	}
	for _, n := range names {
		if s, err := open(m, n, windows.SERVICE_QUERY_CONFIG); err == nil {
			cfg, cerr := s.Config()
			s.Close()
			if cerr == nil && strings.EqualFold(cfg.DisplayName, name) {
				return n, nil
			}
		}
	}
	if sg := Suggest(names, name); len(sg) > 0 {
		return "", fmt.Errorf("%w: %s (có phải ý bạn là: %s?)", ErrNotFound, name, strings.Join(sg, ", "))
	}
	return "", fmt.Errorf("%w: %s", ErrNotFound, name)
}

// waitFor poll mỗi 300ms tới khi service vào trạng thái want hoặc hết thời gian.
func waitFor(s *mgr.Service, want svc.State, wait time.Duration) error {
	deadline := time.Now().Add(wait)
	for {
		st, err := s.Query()
		if err != nil {
			return err
		}
		if st.State == want {
			return nil
		}
		if time.Now().After(deadline) {
			return ErrTimeout
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func change(name string, wait time.Duration, access uint32, act func(*mgr.Service) error, want svc.State) (Service, error) {
	m, err := connect()
	if err != nil {
		return Service{}, err
	}
	defer m.Disconnect()
	real, err := resolve(m, name)
	if err != nil {
		return Service{}, err
	}
	s, err := open(m, real, access|windows.SERVICE_QUERY_STATUS|windows.SERVICE_QUERY_CONFIG)
	if err != nil {
		return Service{}, err
	}
	defer s.Close()
	if st, err := s.Query(); err == nil && st.State == want {
		return describe(s) // đã ở trạng thái mong muốn, không lỗi
	}
	if err := act(s); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) && !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return Service{}, mapErr(err)
	}
	if err := waitFor(s, want, wait); err != nil {
		return Service{}, mapErr(err)
	}
	return describe(s)
}

func Start(name string, wait time.Duration) (Service, error) {
	return change(name, wait, windows.SERVICE_START, func(s *mgr.Service) error { return s.Start() }, svc.Running)
}

func Stop(name string, wait time.Duration) (Service, error) {
	return change(name, wait, windows.SERVICE_STOP, func(s *mgr.Service) error { _, err := s.Control(svc.Stop); return err }, svc.Stopped)
}

// Restart: stop (nếu đang chạy) rồi start.
func Restart(name string, wait time.Duration) (Service, error) {
	if _, err := Stop(name, wait); err != nil {
		return Service{}, err
	}
	return Start(name, wait)
}
