package tcpcheck

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

func TestParseTarget(t *testing.T) {
	tests := []struct {
		in, want string
		ok       bool
	}{
		{"a:1", "a:1", true},
		{"[::1]:80", "[::1]:80", true},
		{"https://a.com", "a.com:443", true},
		{"http://a.com", "a.com:80", true},
		{"mysql://db", "db:3306", true},
		{"redis://h", "h:6379", true},
		{"postgres://h", "h:5432", true},
		{"https://a.com:8443/x", "a.com:8443", true},
		{"a", "", false},
		{"a:", "", false},
		{"a:99999", "", false},
		{"weird://h", "", false},
	}
	for _, tc := range tests {
		got, err := ParseTarget(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseTarget(%q) = %q, %v", tc.in, got, err)
		}
	}
}

func TestCheckAllOrderOpenAndRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	closed, _ := net.Listen("tcp", "127.0.0.1:0")
	closedAddr := closed.Addr().String()
	closed.Close()
	items := CheckAll(context.Background(), []string{closedAddr, ln.Addr().String(), "khongcoport"}, 2*time.Second)
	if len(items) != 3 || items[0].Target != closedAddr || !items[1].Open || items[1].IP != "127.0.0.1" {
		t.Fatalf("%+v", items)
	}
	if items[0].Open || items[0].Error != "từ chối kết nối" {
		t.Errorf("port đóng phải là từ chối kết nối: %+v", items[0])
	}
	if items[2].Error == "" || !strings.Contains(items[2].Error, "host:port") {
		t.Errorf("%+v", items[2])
	}
}

func TestHundredTargetsParallel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	targets := make([]string, 100)
	for i := range targets {
		targets[i] = ln.Addr().String()
	}
	start := time.Now()
	items := CheckAll(context.Background(), targets, 3*time.Second)
	if time.Since(start) > 3*time.Second {
		t.Error("100 đích phải xong trong ~1 timeout")
	}
	for _, it := range items {
		if !it.Open {
			t.Fatalf("%+v", it)
		}
	}
}
