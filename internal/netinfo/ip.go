package netinfo

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Addr struct {
	Iface   string `json:"iface"`
	IP      string `json:"ip"`
	Prefix  int    `json:"prefix"`
	V6      bool   `json:"v6"`
	Virtual bool   `json:"virtual"`
	MAC     string `json:"mac"`
}

type Result struct {
	LAN     []Addr `json:"lan"`
	Primary string `json:"primary"`
	Public  string `json:"public,omitempty"`
}

var virtualWords = []string{"vethernet", "virtualbox", "vmware", "hyper-v", "docker", "wsl", "loopback", "bluetooth", "tailscale", "zerotier"}

// IsVirtual: tên card chứa từ khóa của card ảo (không phân biệt hoa thường).
func IsVirtual(name string) bool {
	l := strings.ToLower(name)
	for _, w := range virtualWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

// LANAddrs liệt kê IP của các card. all=true gồm card ảo, card đang down và địa chỉ link-local.
func LANAddrs(all, includeV6 bool) ([]Addr, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	out := []Addr{}
	for _, ifc := range ifs {
		if ifc.Flags&net.FlagLoopback != 0 || (!all && ifc.Flags&net.FlagUp == 0) {
			continue
		}
		virtual := IsVirtual(ifc.Name)
		if virtual && !all {
			continue
		}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			n, ok := a.(*net.IPNet) // type assertion ≈ instanceof
			if !ok {
				continue
			}
			v6 := n.IP.To4() == nil
			if v6 && !includeV6 {
				continue
			}
			if !all && (n.IP.IsLinkLocalUnicast()) {
				continue
			}
			ones, _ := n.Mask.Size()
			out = append(out, Addr{Iface: ifc.Name, IP: n.IP.String(), Prefix: ones, V6: v6, Virtual: virtual, MAC: ifc.HardwareAddr.String()})
		}
	}
	return out, nil
}

// PrimaryIP: IP dùng để ra Internet. UDP "connect" không gửi gói nào, chỉ để hệ điều hành chọn route.
func PrimaryIP() (string, error) {
	c, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer c.Close()
	return c.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

// publicURLs có thể thay trong test (httptest).
var publicURLs = []string{"https://api.ipify.org", "https://icanhazip.com"}

// PublicIP hỏi dịch vụ ngoài (chỉ gọi khi người dùng bật --public).
func PublicIP(ctx context.Context) (string, error) {
	var last error
	for _, u := range publicURLs {
		ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
		ip, err := fetchIP(ctx2, u)
		cancel()
		if err == nil {
			return ip, nil
		}
		last = err
	}
	return "", last
}

func fetchIP(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(b))
	if net.ParseIP(ip) == nil {
		return "", fmt.Errorf("%s trả về giá trị không phải IP", url)
	}
	return ip, nil
}
