package svc

import (
	"errors"
	"strings"
)

type Service struct {
	Name      string `json:"name"`
	Display   string `json:"display"`
	State     string `json:"state"`
	StartType string `json:"startType"`
	PID       uint32 `json:"pid,omitempty"`
}

var (
	ErrUnsupported = errors.New("chỉ hỗ trợ Windows")
	ErrNotFound    = errors.New("không tìm thấy service")
	ErrTimeout     = errors.New("quá thời gian chờ service đổi trạng thái")
	ErrDependents  = errors.New("service còn service phụ thuộc đang chạy")
	ErrDisabled    = errors.New("service đang bị vô hiệu hóa")
	ErrAccess      = errors.New("không đủ quyền")
)

// devWords: service hay dùng khi dev (khớp chuỗi con, không phân biệt hoa thường).
var devWords = []string{"mysql", "mariadb", "postgres", "redis", "memurai", "mongo", "apache", "httpd", "nginx", "mssql", "sqlserver", "docker", "elasticsearch", "rabbitmq", "php", "laragon"}

// IsDev: tên hoặc display name chứa từ khóa dev.
func IsDev(name, display string) bool {
	l := strings.ToLower(name + "\x00" + display)
	for _, w := range devWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// Matches: chuỗi con của name hoặc display name, không phân biệt hoa thường.
func Matches(name, display, filter string) bool {
	f := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(name), f) || strings.Contains(strings.ToLower(display), f)
}

// StateName đổi mã SERVICE_* của Windows sang tên.
func StateName(state uint32) string {
	switch state {
	case 1:
		return "stopped"
	case 2:
		return "starting"
	case 3:
		return "stopping"
	case 4:
		return "running"
	case 5:
		return "starting" // continue pending
	case 6:
		return "stopping" // pause pending
	case 7:
		return "paused"
	}
	return "unknown"
}

// StartTypeName: 0 boot, 1 system, 2 auto, 3 manual, 4 disabled; auto + delayed → auto-delayed.
func StartTypeName(t uint32, delayed bool) string {
	switch t {
	case 0:
		return "boot"
	case 1:
		return "system"
	case 2:
		if delayed {
			return "auto-delayed"
		}
		return "auto"
	case 3:
		return "manual"
	case 4:
		return "disabled"
	}
	return "unknown"
}

// Suggest trả tối đa 3 tên chứa chuỗi đã nhập (gợi ý khi gõ sai tên service).
func Suggest(names []string, q string) []string {
	var out []string
	l := strings.ToLower(q)
	for _, n := range names {
		if strings.Contains(strings.ToLower(n), l) {
			out = append(out, n)
			if len(out) == 3 {
				break
			}
		}
	}
	return out
}
