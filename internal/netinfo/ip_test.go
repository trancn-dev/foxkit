package netinfo

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsVirtual(t *testing.T) {
	for name, want := range map[string]bool{
		"vEthernet (WSL)": true, "VirtualBox Host-Only": true, "VMware Network Adapter VMnet8": true,
		"Docker Desktop": true, "Wi-Fi": false, "Ethernet": false, "Bluetooth Network Connection": true,
	} {
		if IsVirtual(name) != want {
			t.Errorf("IsVirtual(%q) != %v", name, want)
		}
	}
}

func TestPublicIPWithFakeServer(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("203.0.113.7\n")) }))
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>lỗi</html>")) }))
	defer bad.Close()
	old := publicURLs
	defer func() { publicURLs = old }()

	publicURLs = []string{bad.URL, good.URL} // dịch vụ đầu trả rác → thử dịch vụ sau
	if ip, err := PublicIP(context.Background()); err != nil || ip != "203.0.113.7" {
		t.Fatalf("%q %v", ip, err)
	}
	publicURLs = []string{bad.URL}
	if _, err := PublicIP(context.Background()); err == nil {
		t.Fatal("cần lỗi khi không dịch vụ nào trả IP hợp lệ")
	}
}

func TestLANAddrsDoesNotReturnLoopback(t *testing.T) {
	addrs, err := LANAddrs(true, true)
	if err != nil {
		t.Skip("không liệt kê được card:", err)
	}
	if addrs == nil {
		t.Fatal("phải là slice không nil")
	}
	for _, a := range addrs {
		if net.ParseIP(a.IP).IsLoopback() {
			t.Errorf("không được có loopback: %+v", a)
		}
	}
}
