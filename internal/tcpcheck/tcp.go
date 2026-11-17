package tcpcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Item struct {
	Target string `json:"target"`
	Addr   string `json:"addr"`
	IP     string `json:"ip,omitempty"`
	Open   bool   `json:"open"`
	Ms     int64  `json:"ms"`
	Error  string `json:"error,omitempty"`
}

var schemePorts = map[string]string{"http": "80", "https": "443", "mysql": "3306", "redis": "6379", "postgres": "5432", "postgresql": "5432", "ssh": "22", "ftp": "21", "smtp": "25"}

// ParseTarget chuẩn hóa host:port, [ipv6]:port hoặc URL (suy port từ scheme).
func ParseTarget(s string) (string, error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "://") {
		u, err := url.Parse(s)
		if err != nil || u.Hostname() == "" {
			return "", fmt.Errorf("URL không hợp lệ: %q", s)
		}
		port := u.Port()
		if port == "" {
			port = schemePorts[strings.ToLower(u.Scheme)]
		}
		if port == "" {
			return "", fmt.Errorf("không biết port mặc định của %q, dùng dạng host:port", u.Scheme)
		}
		return net.JoinHostPort(u.Hostname(), port), nil
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil || host == "" || port == "" {
		return "", fmt.Errorf("thiếu port, dùng dạng host:port: %q", s)
	}
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return "", fmt.Errorf("port phải từ 1 đến 65535: %q", port)
	}
	return net.JoinHostPort(host, port), nil
}

const wsaeconnrefused = 10061 // Windows: WSAECONNREFUSED không bằng syscall.ECONNREFUSED nên so thủ công

func describe(err error, timeout time.Duration) string {
	var ne net.Error
	var errno syscall.Errno
	var dns *net.DNSError
	switch {
	case errors.Is(err, os.ErrDeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()):
		return fmt.Sprintf("timeout (%s)", timeout)
	case errors.As(err, &dns):
		return "không phân giải được tên"
	case errors.As(err, &errno) && (errno == syscall.ECONNREFUSED || uintptr(errno) == wsaeconnrefused):
		return "từ chối kết nối"
	}
	return err.Error()
}

// CheckAll kiểm tra song song, giữ đúng thứ tự nhập (mỗi goroutine ghi một ô riêng nên không cần mutex).
func CheckAll(ctx context.Context, targets []string, timeout time.Duration) []Item {
	items := make([]Item, len(targets))
	var wg sync.WaitGroup
	for i, t := range targets {
		items[i].Target = t
		wg.Add(1)
		go func(i int, t string) {
			defer wg.Done()
			addr, err := ParseTarget(t)
			if err != nil {
				items[i].Error = err.Error()
				return
			}
			items[i].Addr = addr
			start := time.Now()
			conn, err := (&net.Dialer{Timeout: timeout}).DialContext(ctx, "tcp", addr)
			items[i].Ms = time.Since(start).Milliseconds()
			if err != nil {
				items[i].Error = describe(err, timeout)
				return
			}
			items[i].Open = true
			items[i].IP, _, _ = net.SplitHostPort(conn.RemoteAddr().String())
			_ = conn.Close() // chỉ cần biết mở được; lỗi đóng không quan trọng
		}(i, t)
	}
	wg.Wait()
	return items
}
