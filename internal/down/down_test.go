package down

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func init() { backoff = func(int) time.Duration { return 5 * time.Millisecond } }

var payload = bytes.Repeat([]byte("0123456789abcdef"), 4096) // 64KB

func sum(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func opts(url, out string) Options {
	return Options{URL: url, Out: out, Resume: true, Retries: 3, Timeout: 2 * time.Second, UserAgent: "foxkit/test"}
}

func noEmit(Event) {}

func kind(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ""
}

func ranged() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		http.ServeContent(w, r, "f.bin", time.Time{}, bytes.NewReader(payload))
	})
}

func TestFullDownloadAndChecksum(t *testing.T) {
	srv := httptest.NewServer(ranged())
	defer srv.Close()
	dir := t.TempDir()
	o := opts(srv.URL+"/file.bin", dir)
	o.SHA256 = sum(payload)
	var events []string
	res, err := Download(context.Background(), o, func(e Event) { events = append(events, e.Event) })
	if err != nil || res.Bytes != int64(len(payload)) || res.SHA256 != sum(payload) || res.Resumed {
		t.Fatalf("%+v %v", res, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "file.bin")); !bytes.Equal(got, payload) {
		t.Fatal("nội dung file sai")
	}
	if events[0] != "start" || events[len(events)-1] != "done" {
		t.Errorf("events: %v", events)
	}
	if _, err := os.Stat(filepath.Join(dir, "file.bin.part")); err == nil {
		t.Error(".part phải được dọn")
	}
}

func TestResumeAfterConnectionDrop(t *testing.T) {
	var calls atomic.Int32
	var sawRange atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		w.Header().Set("ETag", `"v1"`)
		if n == 1 { // lần đầu: báo đủ Content-Length nhưng ngắt giữa chừng
			w.Header().Set("Content-Length", "65536")
			w.WriteHeader(200)
			_, _ = w.Write(payload[:20000])
			w.(http.Flusher).Flush()
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
		sawRange.Store(r.Header.Get("Range"))
		ranged().ServeHTTP(w, r)
	}))
	defer srv.Close()
	dir := t.TempDir()
	o := opts(srv.URL+"/big.bin", dir)
	o.SHA256 = sum(payload)
	res, err := Download(context.Background(), o, noEmit)
	if err != nil || !res.Resumed || res.Bytes != int64(len(payload)) {
		t.Fatalf("%+v %v", res, err)
	}
	if r, _ := sawRange.Load().(string); !strings.HasPrefix(r, "bytes=") {
		t.Errorf("lần thử lại phải gửi Range, got %q", r)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "big.bin")); !bytes.Equal(got, payload) {
		t.Fatal("file sau resume phải giống nguồn")
	}
}

func TestServerIgnoringRangeRestartsFromScratch(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Length", "65536")
		if n == 1 {
			w.WriteHeader(200)
			_, _ = w.Write(payload[:10000])
			w.(http.Flusher).Flush()
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
		_, _ = w.Write(payload) // bỏ qua Range: luôn 200 + toàn bộ
	}))
	defer srv.Close()
	dir := t.TempDir()
	o := opts(srv.URL+"/x.bin", dir)
	o.SHA256 = sum(payload)
	res, err := Download(context.Background(), o, noEmit)
	if err != nil || res.Bytes != int64(len(payload)) {
		t.Fatalf("%+v %v", res, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "x.bin")); !bytes.Equal(got, payload) {
		t.Fatal("không được hỏng file khi server bỏ qua Range")
	}
}

func TestEvilContentDispositionIsSanitized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Disposition", `attachment; filename="..\..\evil.exe"`)
		_, _ = w.Write([]byte("x"))
	}))
	defer srv.Close()
	dir := filepath.Join(t.TempDir(), "dl")
	res, err := Download(context.Background(), opts(srv.URL+"/a", dir+string(os.PathSeparator)), noEmit)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(res.Path) != dir || filepath.Base(res.Path) != "evil.exe" {
		t.Fatalf("tên không an toàn: %s", res.Path)
	}
	for in, want := range map[string]string{`../../x.txt`: "x.txt", `a<b>:c.txt`: "a_b__c.txt", ``: "download", `...`: "download", `C:\Windows\evil.dll`: "evil.dll"} {
		if got := SafeName(in); got != want {
			t.Errorf("SafeName(%q) = %q, cần %q", in, got, want)
		}
	}
}

func TestChecksumMismatchDeletesFile(t *testing.T) {
	srv := httptest.NewServer(ranged())
	defer srv.Close()
	dir := t.TempDir()
	o := opts(srv.URL+"/f.bin", dir)
	o.SHA256 = strings.Repeat("0", 64)
	if _, err := Download(context.Background(), o, noEmit); kind(err) != "checksum" {
		t.Fatalf("%v", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("file/part phải bị xóa, còn %d mục", len(entries))
	}
}

func TestStatusRetryRules(t *testing.T) {
	var c404, c503 atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/404") {
			c404.Add(1)
			http.NotFound(w, r)
			return
		}
		c503.Add(1)
		w.WriteHeader(503)
	}))
	defer srv.Close()
	dir := t.TempDir()
	_, err := Download(context.Background(), opts(srv.URL+"/404", dir), noEmit)
	var e *Error
	if !errors.As(err, &e) || e.Kind != "status" || e.Status != 404 || c404.Load() != 1 {
		t.Fatalf("404 không retry: %v calls=%d", err, c404.Load())
	}
	_, err = Download(context.Background(), opts(srv.URL+"/503", dir), noEmit)
	if !errors.As(err, &e) || e.Status != 503 || c503.Load() != 4 { // 1 + 3 lần thử lại
		t.Fatalf("503 phải retry rồi báo lỗi: %v calls=%d", err, c503.Load())
	}
}

func TestExistsAndForceAndBadURL(t *testing.T) {
	srv := httptest.NewServer(ranged())
	defer srv.Close()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "f.bin"), []byte("cũ"), 0o644)
	if _, err := Download(context.Background(), opts(srv.URL+"/f.bin", dir), noEmit); kind(err) != "exists" {
		t.Fatalf("%v", err)
	}
	o := opts(srv.URL+"/f.bin", dir)
	o.Force = true
	if res, err := Download(context.Background(), o, noEmit); err != nil || res.Bytes != int64(len(payload)) {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := Download(context.Background(), opts("ftp://x/y", dir), noEmit); kind(err) != "url" {
		t.Errorf("%v", err)
	}
}

func TestCancelKeepsPart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		w.Header().Set("Content-Length", "65536")
		w.WriteHeader(200)
		_, _ = w.Write(payload[:5000])
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if _, err := Download(ctx, opts(srv.URL+"/c.bin", dir), noEmit); err == nil {
		t.Fatal("cần lỗi khi hủy")
	}
	if st, err := os.Stat(filepath.Join(dir, "c.bin.part")); err != nil || st.Size() == 0 {
		t.Errorf("phải giữ .part để tiếp tục: %v", err)
	}
}

func TestBarAndETA(t *testing.T) {
	if got := Bar(50, 100, 10); got != "[█████░░░░░]  50%" {
		t.Errorf("%q", got)
	}
	if Bar(1, 0, 10) != "" {
		t.Error("không biết tổng thì không vẽ thanh")
	}
	if ETA(0, 100, 10) != "0m10s" || ETA(0, 100, 0) != "" {
		t.Errorf("ETA: %q", ETA(0, 100, 10))
	}
}

func TestParseHeaders(t *testing.T) {
	h, err := ParseHeaders([]string{"Authorization: Bearer x", "X-A:b"})
	if err != nil || h.Get("Authorization") != "Bearer x" || h.Get("X-A") != "b" {
		t.Fatalf("%v %v", h, err)
	}
	if _, err := ParseHeaders([]string{"khongcohaicham"}); err == nil {
		t.Error("thiếu : phải lỗi")
	}
}
