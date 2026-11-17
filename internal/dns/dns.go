package dns

import (
	"context"
	"errors"
	"net"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"foxkit/internal/winexec"
)

type MX struct {
	Host string `json:"host"`
	Pref uint16 `json:"pref"`
}

type Records struct {
	A     []string `json:"A,omitempty"`
	AAAA  []string `json:"AAAA,omitempty"`
	CNAME string   `json:"CNAME,omitempty"`
	MX    []MX     `json:"MX,omitempty"`
	TXT   []string `json:"TXT,omitempty"`
	NS    []string `json:"NS,omitempty"`
}

type Result struct {
	Host    string            `json:"host"`
	Server  string            `json:"server"`
	Records Records           `json:"records"`
	Errors  map[string]string `json:"errors"`
	Ms      int64             `json:"ms"`
}

var AllTypes = []string{"A", "AAAA", "CNAME", "MX", "TXT", "NS"}

// NormalizeHost: người dùng hay dán cả URL → bỏ scheme, path, port.
func NormalizeHost(s string) string {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "://") {
		if u, err := url.Parse(s); err == nil && u.Hostname() != "" {
			return u.Hostname()
		}
	}
	s = strings.SplitN(s, "/", 2)[0]
	if h, _, err := net.SplitHostPort(s); err == nil {
		return h
	}
	return s
}

func resolver(server string) *net.Resolver {
	if server == "" {
		return net.DefaultResolver
	}
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "udp", net.JoinHostPort(server, "53"))
	}}
}

func describe(err error) string {
	var de *net.DNSError
	if errors.As(err, &de) && de.IsNotFound {
		return "không có bản ghi"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "quá thời gian chờ"
	}
	return err.Error()
}

// Lookup tra các loại bản ghi song song (mỗi loại timeout 5s). Lỗi từng loại nằm trong Errors.
func Lookup(ctx context.Context, host string, types []string, server string) Result {
	start := time.Now()
	res := Result{Host: host, Server: "system", Errors: map[string]string{}}
	if server != "" {
		res.Server = server
	}
	r := resolver(server)
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, t := range types {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			var err error
			set := func(f func()) { mu.Lock(); f(); mu.Unlock() }
			switch t {
			case "A", "AAAA":
				network := "ip4"
				if t == "AAAA" {
					network = "ip6"
				}
				var ips []net.IP
				if ips, err = r.LookupIP(ctx, network, host); err == nil {
					s := make([]string, len(ips))
					for i, ip := range ips {
						s[i] = ip.String()
					}
					sort.Strings(s)
					set(func() {
						if t == "A" {
							res.Records.A = s
						} else {
							res.Records.AAAA = s
						}
					})
				}
			case "CNAME":
				var c string
				if c, err = r.LookupCNAME(ctx, host); err == nil {
					if strings.EqualFold(strings.TrimSuffix(c, "."), host) { // trả về chính host = không có CNAME
						err = &net.DNSError{Err: "no such record", Name: host, IsNotFound: true}
					} else {
						set(func() { res.Records.CNAME = c })
					}
				}
			case "MX":
				var mx []*net.MX
				if mx, err = r.LookupMX(ctx, host); err == nil {
					out := make([]MX, len(mx))
					for i, m := range mx {
						out[i] = MX{Host: m.Host, Pref: m.Pref}
					}
					set(func() { res.Records.MX = out })
				}
			case "TXT":
				var txt []string
				if txt, err = r.LookupTXT(ctx, host); err == nil {
					set(func() { res.Records.TXT = txt })
				}
			case "NS":
				var ns []*net.NS
				if ns, err = r.LookupNS(ctx, host); err == nil {
					out := make([]string, len(ns))
					for i, n := range ns {
						out[i] = n.Host
					}
					set(func() { res.Records.NS = out })
				}
			}
			if err != nil {
				set(func() { res.Errors[t] = describe(err) })
			}
		}(t)
	}
	wg.Wait()
	res.Ms = time.Since(start).Milliseconds()
	return res
}

// Flush: ipconfig /flushdns. Chỉ kiểm exit code, không parse chữ (bị dịch theo ngôn ngữ).
func Flush(ctx context.Context) error {
	_, err := winexec.Run(ctx, 15*time.Second, "ipconfig", "/flushdns")
	return err
}
