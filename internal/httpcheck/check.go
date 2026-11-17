package httpcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

type Timing struct {
	DNSMs     int64 `json:"dnsMs"`
	ConnectMs int64 `json:"connectMs"`
	TLSMs     int64 `json:"tlsMs"`
	TTFBMs    int64 `json:"ttfbMs"`
	TotalMs   int64 `json:"totalMs"`
}

type TLSInfo struct {
	Subject  string    `json:"subject"`
	Issuer   string    `json:"issuer"`
	Expires  time.Time `json:"expires"`
	DaysLeft int       `json:"daysLeft"`
	Version  string    `json:"version"`
}

type Hop struct {
	URL    string `json:"url"`
	Status int    `json:"status"`
}

type Result struct {
	URL       string            `json:"url"`
	FinalURL  string            `json:"finalUrl"`
	Status    int               `json:"status"`
	Proto     string            `json:"proto"`
	Timing    Timing            `json:"timing"`
	Redirects []Hop             `json:"redirects"`
	Headers   map[string]string `json:"headers"`
	Bytes     int64             `json:"bytes"`
	TLS       *TLSInfo          `json:"tls,omitempty"`
	Body      string            `json:"body,omitempty"`
}

type Options struct {
	URL       string
	Method    string
	Headers   []string // "Key: Value"
	Data      string
	Timeout   time.Duration
	NoFollow  bool
	Insecure  bool
	WantBody  bool
	UserAgent string
}

// Error phân loại lỗi để cmd map sang mã lỗi: Kind = timeout | connect | tls | redirects.
type Error struct {
	Kind string
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

var errTooMany = errors.New("quá 10 lần chuyển hướng")

// NormalizeURL thêm https:// nếu thiếu scheme và kiểm tra có host.
func NormalizeURL(s string) (string, error) {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("URL không hợp lệ: %q", s)
	}
	return u.String(), nil
}

// ParseHeader tách "Key: Value".
func ParseHeader(h string) (k, v string, err error) {
	i := strings.Index(h, ":")
	if i <= 0 {
		return "", "", fmt.Errorf("header thiếu dấu ':': %q", h)
	}
	return strings.TrimSpace(h[:i]), strings.TrimSpace(h[i+1:]), nil
}

// tracer ghi mốc thời gian của request đang chạy; reset khi bắt đầu hop mới để chỉ đo request cuối.
type tracer struct {
	start               time.Time
	dnsStart, dnsDone   time.Time
	connStart, connDone time.Time
	tlsStart, tlsDone   time.Time
	wrote, firstByte    time.Time
}

func (t *tracer) reset() { *t = tracer{start: time.Now()} }

func ms(a, b time.Time) int64 {
	if a.IsZero() || b.IsZero() || b.Before(a) {
		return 0
	}
	return b.Sub(a).Milliseconds()
}

func classify(err error) error {
	var uerr *url.Error
	switch {
	case errors.Is(err, errTooMany):
		return &Error{"redirects", errTooMany}
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &uerr) && uerr.Timeout()):
		return &Error{"timeout", err}
	}
	var cve *tls.CertificateVerificationError
	var ua x509.UnknownAuthorityError
	var hn x509.HostnameError
	var ci x509.CertificateInvalidError
	if errors.As(err, &cve) || errors.As(err, &ua) || errors.As(err, &hn) || errors.As(err, &ci) {
		return &Error{"tls", err}
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return &Error{"timeout", err}
	}
	return &Error{"connect", err}
}

func Check(ctx context.Context, opt Options) (Result, error) {
	method := strings.ToUpper(opt.Method)
	if method == "" {
		method = http.MethodGet
	}
	tr := http.DefaultTransport.(*http.Transport).Clone() // giữ proxy từ biến môi trường
	if opt.Insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // người dùng chủ động bật --insecure cho dev local
	}
	res := Result{URL: opt.URL, Redirects: []Hop{}, Headers: map[string]string{}}
	trace := &tracer{}
	client := &http.Client{
		Timeout:   opt.Timeout,
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if opt.NoFollow {
				return http.ErrUseLastResponse
			}
			if len(via) >= 10 {
				return errTooMany
			}
			res.Redirects = append(res.Redirects, Hop{URL: via[len(via)-1].URL.String(), Status: req.Response.StatusCode})
			trace.reset()
			return nil
		},
	}
	var body io.Reader
	if opt.Data != "" {
		body = strings.NewReader(opt.Data)
	}
	ct := &httptrace.ClientTrace{
		DNSStart:             func(httptrace.DNSStartInfo) { trace.dnsStart = time.Now() },
		DNSDone:              func(httptrace.DNSDoneInfo) { trace.dnsDone = time.Now() },
		ConnectStart:         func(_, _ string) { trace.connStart = time.Now() },
		ConnectDone:          func(_, _ string, _ error) { trace.connDone = time.Now() },
		TLSHandshakeStart:    func() { trace.tlsStart = time.Now() },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { trace.tlsDone = time.Now() },
		WroteRequest:         func(httptrace.WroteRequestInfo) { trace.wrote = time.Now() },
		GotFirstResponseByte: func() { trace.firstByte = time.Now() },
	}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, ct), method, opt.URL, body)
	if err != nil {
		return res, err
	}
	for _, h := range opt.Headers {
		k, v, herr := ParseHeader(h)
		if herr != nil {
			return res, herr
		}
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", opt.UserAgent)
	}
	trace.reset()
	resp, err := client.Do(req)
	if err != nil {
		return res, classify(err)
	}
	defer resp.Body.Close()

	res.FinalURL = resp.Request.URL.String()
	res.Status = resp.StatusCode
	res.Proto = resp.Proto
	for k, v := range resp.Header {
		res.Headers[strings.ToLower(k)] = strings.Join(v, ", ")
	}
	if opt.WantBody {
		head, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10)) // lỗi đọc giữa chừng: giữ phần đã có
		rest, _ := io.Copy(io.Discard, resp.Body)
		res.Bytes = int64(len(head)) + rest
		res.Body = string(head)
		if rest > 0 {
			res.Body += "\n… (đã cắt, chỉ hiện 64KB đầu)"
		}
	} else {
		res.Bytes, _ = io.Copy(io.Discard, resp.Body)
	}
	end := time.Now()
	res.Timing = Timing{
		DNSMs:     ms(trace.dnsStart, trace.dnsDone),
		ConnectMs: ms(trace.connStart, trace.connDone),
		TLSMs:     ms(trace.tlsStart, trace.tlsDone),
		TTFBMs:    ms(trace.wrote, trace.firstByte),
		TotalMs:   ms(trace.start, end),
	}
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		c := resp.TLS.PeerCertificates[0]
		res.TLS = &TLSInfo{
			Subject:  firstNonEmpty(c.Subject.CommonName, first(c.DNSNames)),
			Issuer:   firstNonEmpty(c.Issuer.CommonName, first(c.Issuer.Organization)),
			Expires:  c.NotAfter,
			DaysLeft: int(time.Until(c.NotAfter).Hours() / 24),
			Version:  tls.VersionName(resp.TLS.Version),
		}
	}
	return res, nil
}

func first(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
