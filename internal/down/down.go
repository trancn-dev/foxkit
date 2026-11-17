package down

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type Options struct {
	URL       string
	Out       string // thư mục hoặc đường dẫn file; rỗng = thư mục hiện tại
	Headers   http.Header
	Resume    bool
	Force     bool
	Retries   int
	Timeout   time.Duration // timeout kết nối và chờ header, không phải cả lần tải
	SHA256    string
	UserAgent string
}

type Event struct {
	Event       string `json:"event"` // start | progress | retry | done
	Bytes       int64  `json:"bytes,omitempty"`
	Total       int64  `json:"total,omitempty"`
	BytesPerSec int64  `json:"bytesPerSec,omitempty"`
	Path        string `json:"path,omitempty"`
	Attempt     int    `json:"attempt,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	DurationMs  int64  `json:"durationMs,omitempty"`
}

type Result struct {
	Path       string `json:"path"`
	Bytes      int64  `json:"bytes"`
	DurationMs int64  `json:"durationMs"`
	SHA256     string `json:"sha256"`
	Resumed    bool   `json:"resumed"`
}

// Error: Kind = url | exists | status | checksum | timeout | connect | disk.
type Error struct {
	Kind   string
	Status int
	Err    error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// backoff chờ giữa các lần thử lại (1s, 2s, 4s...); test đặt lại cho ngắn.
var backoff = func(attempt int) time.Duration { return time.Second << (attempt - 1) }

var invalidChars = `<>:"/\|?*`

// SafeName làm sạch tên file nhận từ bên ngoài: bỏ mọi thành phần đường dẫn, thay ký tự không hợp lệ trên Windows,
// để server độc hại không ghi ra ngoài thư mục đích (path traversal).
func SafeName(s string) string {
	if i := strings.LastIndexAny(s, `/\`); i >= 0 {
		s = s[i+1:]
	}
	s = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(invalidChars, r) {
			return '_'
		}
		return r
	}, s)
	s = strings.Trim(s, ". ")
	if s == "" {
		return "download"
	}
	return s
}

func nameFromURL(u *url.URL) string { return SafeName(filepath.Base(u.Path)) }

func nameFromDisposition(h string) string {
	if h == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(h)
	if err != nil || params["filename"] == "" {
		return ""
	}
	return SafeName(params["filename"])
}

type meta struct {
	ETag         string `json:"etag,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
}

func readMeta(p string) meta {
	var m meta
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &m) // meta hỏng thì coi như không có → tải lại từ đầu
	}
	return m
}

func writeMeta(p string, m meta) {
	if b, err := json.Marshal(m); err == nil {
		_ = os.WriteFile(p, b, 0o644) // meta chỉ để resume an toàn; không ghi được thì lần sau tải lại từ đầu
	}
}

func classify(err error) error {
	var ne net.Error
	switch {
	case errors.Is(err, syscall.Errno(112)) || errors.Is(err, syscall.Errno(39)): // ERROR_DISK_FULL, ERROR_HANDLE_DISK_FULL
		return &Error{Kind: "disk", Err: err}
	case errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()):
		return &Error{Kind: "timeout", Err: err}
	}
	return &Error{Kind: "connect", Err: err}
}

func retryable(err error) bool {
	var e *Error
	if !errors.As(err, &e) {
		return false
	}
	switch e.Kind {
	case "connect", "timeout":
		return true
	case "status":
		return e.Status >= 500 || e.Status == 408 || e.Status == 429
	}
	return false
}

// Download tải url về Out, resume từ file .part khi server hỗ trợ Range, kiểm sha256, thử lại có backoff.
func Download(ctx context.Context, opt Options, emit func(Event)) (Result, error) {
	start := time.Now()
	u, err := url.Parse(opt.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return Result{}, &Error{Kind: "url", Err: fmt.Errorf("URL không hợp lệ: %q", opt.URL)}
	}
	dir, name := ".", nameFromURL(u)
	if opt.Out != "" {
		if st, err := os.Stat(opt.Out); err == nil && st.IsDir() || strings.HasSuffix(opt.Out, `\`) || strings.HasSuffix(opt.Out, "/") {
			dir = opt.Out
		} else {
			dir, name = filepath.Dir(opt.Out), filepath.Base(opt.Out)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, classify(err)
	}
	final := filepath.Join(dir, name)
	if _, err := os.Stat(final); err == nil && !opt.Force {
		return Result{}, &Error{Kind: "exists", Err: fmt.Errorf("file đã tồn tại: %s", final)}
	}
	part, metaPath := final+".part", final+".part.meta"

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = (&net.Dialer{Timeout: opt.Timeout}).DialContext
	tr.ResponseHeaderTimeout = opt.Timeout
	client := &http.Client{Transport: tr} // không đặt Client.Timeout: đó là timeout cả lần tải

	var resumed bool
	var lastErr error
	for attempt := 0; attempt <= opt.Retries; attempt++ {
		if attempt > 0 {
			emit(Event{Event: "retry", Attempt: attempt, Path: final})
			select {
			case <-time.After(backoff(attempt)):
			case <-ctx.Done():
				return Result{}, ctx.Err()
			}
		}
		res, wasResumed, err := fetch(ctx, client, opt, u, final, part, metaPath, &name, emit)
		resumed = resumed || wasResumed
		if err == nil {
			res.Resumed = resumed
			res.DurationMs = time.Since(start).Milliseconds()
			emit(Event{Event: "done", Path: res.Path, Bytes: res.Bytes, SHA256: res.SHA256, DurationMs: res.DurationMs})
			return res, nil
		}
		if ctx.Err() != nil {
			return Result{}, ctx.Err() // giữ nguyên .part để lần sau tiếp tục
		}
		lastErr = err
		if !retryable(err) {
			return Result{}, err
		}
	}
	return Result{}, lastErr
}

func fetch(ctx context.Context, client *http.Client, opt Options, u *url.URL, final, part, metaPath string, name *string, emit func(Event)) (Result, bool, error) {
	var have int64
	m := readMeta(metaPath)
	if st, err := os.Stat(part); err == nil && opt.Resume && (m.ETag != "" || m.LastModified != "") {
		have = st.Size()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Result{}, false, &Error{Kind: "url", Err: err}
	}
	for k, vs := range opt.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	req.Header.Set("User-Agent", opt.UserAgent)
	if have > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", have))
		if m.ETag != "" {
			req.Header.Set("If-Range", m.ETag)
		} else {
			req.Header.Set("If-Range", m.LastModified)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return Result{}, false, classify(err)
	}
	defer resp.Body.Close()

	resumed := false
	var total int64
	switch {
	case resp.StatusCode == http.StatusPartialContent && have > 0:
		from, tot, err := parseContentRange(resp.Header.Get("Content-Range"))
		if err != nil || from != have {
			// Content-Range không khớp vị trí đã có: bỏ .part, tải lại từ đầu ở lần thử sau
			_ = os.Remove(part)
			return Result{}, false, &Error{Kind: "connect", Err: errors.New("server trả Content-Range không khớp")}
		}
		total, resumed = tot, true
	case resp.StatusCode == http.StatusOK:
		have, total = 0, resp.ContentLength // server bỏ qua Range hoặc tải mới: ghi lại từ đầu
	case resp.StatusCode == http.StatusRequestedRangeNotSatisfiable:
		_ = os.Remove(part)
		_ = os.Remove(metaPath)
		return Result{}, false, &Error{Kind: "connect", Err: errors.New("416: .part không dùng được, tải lại từ đầu")}
	default:
		return Result{}, false, &Error{Kind: "status", Status: resp.StatusCode, Err: fmt.Errorf("server trả %d", resp.StatusCode)}
	}
	if n := nameFromDisposition(resp.Header.Get("Content-Disposition")); n != "" && (opt.Out == "" || isDirOut(opt.Out)) {
		*name = n
	}
	target := filepath.Join(filepath.Dir(final), *name)
	if target != final {
		if _, err := os.Stat(target); err == nil && !opt.Force {
			return Result{}, false, &Error{Kind: "exists", Err: fmt.Errorf("file đã tồn tại: %s", target)}
		}
	}
	writeMeta(metaPath, meta{ETag: resp.Header.Get("ETag"), LastModified: resp.Header.Get("Last-Modified")})

	flag := os.O_CREATE | os.O_WRONLY
	if resumed {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	f, err := os.OpenFile(part, flag, 0o644)
	if err != nil {
		return Result{}, false, classify(err)
	}
	emit(Event{Event: "start", Total: total, Bytes: have, Path: target})

	var sp speedometer
	written := have
	last := time.Now()
	buf := make([]byte, 64<<10)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return Result{}, resumed, classify(werr)
			}
			written += int64(n)
			if now := time.Now(); now.Sub(last) >= 250*time.Millisecond {
				sp.add(now, written)
				emit(Event{Event: "progress", Bytes: written, Total: total, BytesPerSec: sp.rate(), Path: target})
				last = now
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return Result{}, resumed, classify(rerr)
		}
	}
	if err := f.Close(); err != nil {
		return Result{}, resumed, classify(err)
	}
	if total > 0 && written != total {
		return Result{}, resumed, &Error{Kind: "connect", Err: fmt.Errorf("tải thiếu: %d/%d byte", written, total)}
	}
	emit(Event{Event: "progress", Bytes: written, Total: total, BytesPerSec: sp.rate(), Path: target})

	sum, err := hashFile(part)
	if err != nil {
		return Result{}, resumed, classify(err)
	}
	if opt.SHA256 != "" && !strings.EqualFold(sum, strings.TrimSpace(opt.SHA256)) {
		_ = os.Remove(part)
		_ = os.Remove(metaPath)
		return Result{}, resumed, &Error{Kind: "checksum", Err: fmt.Errorf("sha256 không khớp: nhận %s", sum)}
	}
	if opt.Force {
		_ = os.Remove(target) // Rename không ghi đè trên Windows nếu đích còn tồn tại
	}
	if err := os.Rename(part, target); err != nil {
		return Result{}, resumed, classify(err)
	}
	_ = os.Remove(metaPath)
	return Result{Path: target, Bytes: written, SHA256: sum}, resumed, nil
}

// parseContentRange đọc "bytes 20000-65535/65536" → (20000, 65536). Tổng "*" (không biết) → 0.
func parseContentRange(h string) (from, total int64, err error) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(h), "bytes ")
	if !ok {
		return 0, 0, fmt.Errorf("Content-Range sai: %q", h)
	}
	span, tot, ok := strings.Cut(rest, "/")
	first, _, ok2 := strings.Cut(span, "-")
	if !ok || !ok2 {
		return 0, 0, fmt.Errorf("Content-Range sai: %q", h)
	}
	if from, err = strconv.ParseInt(first, 10, 64); err != nil {
		return 0, 0, err
	}
	if tot != "*" {
		if total, err = strconv.ParseInt(tot, 10, 64); err != nil {
			return 0, 0, err
		}
	}
	return from, total, nil
}

func isDirOut(out string) bool {
	st, err := os.Stat(out)
	return err == nil && st.IsDir() || strings.HasSuffix(out, `\`) || strings.HasSuffix(out, "/")
}

func hashFile(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ParseHeaders đổi các "Key: Value" thành http.Header.
func ParseHeaders(list []string) (http.Header, error) {
	h := http.Header{}
	for _, s := range list {
		k, v, ok := strings.Cut(s, ":")
		if !ok || strings.TrimSpace(k) == "" {
			return nil, fmt.Errorf("header thiếu dấu ':': %q", s)
		}
		h.Add(strings.TrimSpace(k), strings.TrimSpace(v))
	}
	return h, nil
}
