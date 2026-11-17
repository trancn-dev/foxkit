package serve

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"
)

func site(t *testing.T) string {
	d := t.TempDir()
	for name, body := range map[string]string{"index.html": "INDEX", "assets/app.js": "JS", ".env": "SECRET=1", "empty/x.txt": "x"} {
		p := filepath.Join(d, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return d
}

func get(t *testing.T, srv *httptest.Server, method, p string) (int, string, http.Header) {
	t.Helper()
	req, _ := http.NewRequest(method, srv.URL+p, nil)
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b), resp.Header
}

func TestStaticAndDotfiles(t *testing.T) {
	srv := httptest.NewServer(Handler(Options{Dir: site(t), NoCache: true}, nil))
	defer srv.Close()
	if c, b, h := get(t, srv, "GET", "/"); c != 200 || b != "INDEX" || h.Get("Cache-Control") != "no-store" {
		t.Errorf("/: %d %q %v", c, b, h)
	}
	if c, b, _ := get(t, srv, "GET", "/assets/app.js"); c != 200 || b != "JS" {
		t.Errorf("app.js: %d %q", c, b)
	}
	for _, p := range []string{"/.env", "/%2eenv", "/assets/../.env"} {
		if c, b, _ := get(t, srv, "GET", p); c == 200 || b == "SECRET=1" {
			t.Errorf("%s không được lộ file ẩn: %d", p, c)
		}
	}
	if c, _, _ := get(t, srv, "GET", "/assets/"); c != 404 {
		t.Errorf("không --list: /assets/ phải 404, got %d", c)
	}
	if c, _, _ := get(t, srv, "GET", "/nope.js"); c != 404 {
		t.Errorf("thiếu file: %d", c)
	}
}

func TestListEnabled(t *testing.T) {
	srv := httptest.NewServer(Handler(Options{Dir: site(t), List: true}, nil))
	defer srv.Close()
	if c, b, _ := get(t, srv, "GET", "/assets/"); c != 200 || b == "" {
		t.Errorf("--list: %d %q", c, b)
	}
}

func TestSPA(t *testing.T) {
	srv := httptest.NewServer(Handler(Options{Dir: site(t), SPA: true}, nil))
	defer srv.Close()
	if c, b, _ := get(t, srv, "GET", "/products/12"); c != 200 || b != "INDEX" {
		t.Errorf("route SPA: %d %q", c, b)
	}
	if c, _, _ := get(t, srv, "GET", "/missing.js"); c != 404 {
		t.Errorf("file có đuôi mà thiếu phải 404, got %d", c)
	}
	if c, b, _ := get(t, srv, "GET", "/assets/app.js"); c != 200 || b != "JS" {
		t.Errorf("file thật vẫn phải phục vụ: %d %q", c, b)
	}
}

func TestCORS(t *testing.T) {
	srv := httptest.NewServer(Handler(Options{Dir: site(t), CORS: true}, nil))
	defer srv.Close()
	if c, _, h := get(t, srv, "GET", "/"); c != 200 || h.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("%d %v", c, h)
	}
	if c, _, _ := get(t, srv, "OPTIONS", "/"); c != 204 {
		t.Errorf("OPTIONS = %d", c)
	}
}

func TestLoggingRecordsStatus(t *testing.T) {
	var mu sync.Mutex
	var evs []Event
	srv := httptest.NewServer(Handler(Options{Dir: site(t)}, func(e Event) { mu.Lock(); evs = append(evs, e); mu.Unlock() }))
	defer srv.Close()
	get(t, srv, "GET", "/nope.js")
	get(t, srv, "GET", "/")
	mu.Lock()
	defer mu.Unlock()
	if len(evs) != 2 || evs[0].Status != 404 || evs[1].Status != 200 || evs[1].Bytes != 5 || evs[0].Path != "/nope.js" || evs[0].Method != "GET" {
		t.Fatalf("%+v", evs)
	}
}

func TestRunPicksNextPortAndStopsGracefully(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	port := busy.Addr().(*net.TCPAddr).Port
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan Event, 1)
	stopped := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{Dir: site(t), Host: "127.0.0.1", Port: port}, func(e Event) {
			switch e.Event {
			case "start":
				started <- e
			case "stop":
				close(stopped)
			}
		})
	}()
	var ev Event
	select {
	case ev = <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("server không start")
	}
	if len(ev.URLs) == 0 || ev.URLs[0] == "http://localhost:"+strconv.Itoa(port) {
		t.Fatalf("port bận phải tự nhảy sang port kế: %v", ev.URLs)
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("Shutdown quá lâu")
	}
	select {
	case <-stopped:
	default:
		t.Error("thiếu sự kiện stop")
	}
}
