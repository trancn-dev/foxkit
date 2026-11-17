package httpcheck

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func opts(url string) Options {
	return Options{URL: url, Timeout: 3 * time.Second, UserAgent: "foxkit/test"}
}

func kind(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ""
}

func TestBasicHeadersAndBytes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "a")
		w.Header().Add("X-Test", "b")
		if r.Header.Get("User-Agent") != "foxkit/test" {
			t.Errorf("UA = %q", r.Header.Get("User-Agent"))
		}
		_, _ = w.Write([]byte("hello"))
	}))
	defer srv.Close()
	o := opts(srv.URL)
	o.WantBody = true
	r, err := Check(context.Background(), o)
	if err != nil || r.Status != 200 || r.Bytes != 5 || r.Body != "hello" || r.Headers["x-test"] != "a, b" || r.TLS != nil || r.Redirects == nil {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestRedirectChain(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/b", http.StatusMovedPermanently) })
	mux.HandleFunc("/b", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/c", http.StatusFound) })
	mux.HandleFunc("/c", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	r, err := Check(context.Background(), opts(srv.URL+"/a"))
	if err != nil || r.Status != 200 || len(r.Redirects) != 2 || r.Redirects[0].Status != 301 || r.Redirects[1].Status != 302 || !strings.HasSuffix(r.FinalURL, "/c") {
		t.Fatalf("%+v %v", r, err)
	}
	o := opts(srv.URL + "/a")
	o.NoFollow = true
	r, err = Check(context.Background(), o)
	if err != nil || r.Status != 301 || len(r.Redirects) != 0 {
		t.Fatalf("no-follow: %+v %v", r, err)
	}
}

func TestTooManyRedirects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, r.URL.Path, http.StatusFound) }))
	defer srv.Close()
	if _, err := Check(context.Background(), opts(srv.URL)); kind(err) != "redirects" {
		t.Fatalf("got %v", err)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { time.Sleep(2 * time.Second) }))
	defer srv.Close()
	o := opts(srv.URL)
	o.Timeout = 300 * time.Millisecond
	if _, err := Check(context.Background(), o); kind(err) != "timeout" {
		t.Fatalf("got %v", err)
	}
}

func TestTLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("x")) }))
	defer srv.Close()
	if _, err := Check(context.Background(), opts(srv.URL)); kind(err) != "tls" {
		t.Fatalf("cert tự ký không --insecure phải là tls, got %v", err)
	}
	o := opts(srv.URL)
	o.Insecure = true
	r, err := Check(context.Background(), o)
	if err != nil || r.TLS == nil || r.TLS.Version == "" || r.TLS.Expires.IsZero() {
		t.Fatalf("%+v %v", r, err)
	}
}

func TestConnectRefused(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	if _, err := Check(context.Background(), opts(url)); kind(err) != "connect" {
		t.Fatalf("got %v", err)
	}
}

func TestNormalizeAndHeader(t *testing.T) {
	if u, err := NormalizeURL("example.com/x"); err != nil || u != "https://example.com/x" {
		t.Errorf("%q %v", u, err)
	}
	for _, bad := range []string{"", "ftp://a.com", "https://"} {
		if _, err := NormalizeURL(bad); err == nil {
			t.Errorf("%q phải lỗi", bad)
		}
	}
	if k, v, err := ParseHeader("X-A:  b: c"); err != nil || k != "X-A" || v != "b: c" {
		t.Errorf("%q %q %v", k, v, err)
	}
	if _, _, err := ParseHeader("khongcodauhaichấm"); err == nil {
		t.Error("thiếu : phải lỗi")
	}
}
