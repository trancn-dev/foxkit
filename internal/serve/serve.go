package serve

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"foxkit/internal/netinfo"
)

type Options struct {
	Dir, Host                string
	Port                     int
	SPA, CORS, NoCache, List bool
}

type Event struct {
	Event  string   `json:"event"`
	Time   string   `json:"time"`
	URLs   []string `json:"urls,omitempty"`
	Method string   `json:"method,omitempty"`
	Path   string   `json:"path,omitempty"`
	Status int      `json:"status,omitempty"`
	Bytes  int64    `json:"bytes,omitempty"`
	Ms     int64    `json:"ms,omitempty"`
	Remote string   `json:"remote,omitempty"`
}

var ErrNoPort = errors.New("20 port liên tiếp đều bận")

func now() string { return time.Now().Format(time.RFC3339) }

// recorder bọc ResponseWriter để ghi lại status và số byte (embed struct để ghi đè WriteHeader/Write).
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int64
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *recorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytes += int64(n)
	return n, err
}

func logging(next http.Handler, emit func(Event)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if emit != nil {
			host, _, _ := net.SplitHostPort(r.RemoteAddr)
			emit(Event{Event: "request", Time: now(), Method: r.Method, Path: r.URL.Path, Status: rec.status, Bytes: rec.bytes, Ms: time.Since(start).Milliseconds(), Remote: host})
		}
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "*")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

// noDotfiles: không bao giờ phục vụ file/thư mục bắt đầu bằng "." (.env, .git).
func noDotfiles(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, seg := range strings.Split(r.URL.Path, "/") {
			if strings.HasPrefix(seg, ".") && seg != "." && seg != ".." {
				http.NotFound(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// noList: đường dẫn kết thúc "/" mà không có index.html → 404 (không liệt kê thư mục).
func noList(root string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path.Clean(r.URL.Path)), "index.html")); err != nil {
				http.NotFound(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// spa: đường dẫn không tồn tại và không có đuôi file → trả index.html; thiếu file có đuôi (/app.js) vẫn 404 thật.
func spa(root string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil && path.Ext(p) == "" {
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			next.ServeHTTP(w, r2)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Handler dựng chuỗi middleware: logging(cors(noCache(noDotfiles(noList(spa(fs)))))). emit có thể nil.
func Handler(opt Options, emit func(Event)) http.Handler {
	root, err := filepath.Abs(opt.Dir)
	if err != nil {
		root = opt.Dir
	}
	var h http.Handler = http.FileServer(http.Dir(root)) // http.Dir đã chặn ".." thoát khỏi root
	if opt.SPA {
		h = spa(root, h)
	}
	if !opt.List {
		h = noList(root, h)
	}
	h = noDotfiles(h)
	if opt.NoCache {
		h = noCache(h)
	}
	if opt.CORS {
		h = cors(h)
	}
	return logging(h, emit)
}

func listen(host string, port int) (net.Listener, int, error) {
	tries := 21
	if port == 0 {
		tries = 1
	}
	var last error
	for i := 0; i < tries; i++ {
		ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port+i)))
		if err == nil {
			return ln, ln.Addr().(*net.TCPAddr).Port, nil
		}
		last = err
	}
	if port == 0 {
		return nil, 0, last
	}
	return nil, 0, fmt.Errorf("%w (%v)", ErrNoPort, last)
}

// Run chạy server tới khi ctx hủy, rồi Shutdown gọn (timeout 5s) và phát sự kiện stop.
func Run(ctx context.Context, opt Options, emit func(Event)) error {
	ln, port, err := listen(opt.Host, opt.Port)
	if err != nil {
		return err
	}
	urls := []string{fmt.Sprintf("http://localhost:%d", port)}
	if opt.Host == "0.0.0.0" || opt.Host == "::" || opt.Host == "" {
		if addrs, err := netinfo.LANAddrs(false, false); err == nil {
			for _, a := range addrs {
				urls = append(urls, fmt.Sprintf("http://%s:%d", a.IP, port))
			}
		}
	}
	srv := &http.Server{Handler: Handler(opt, emit), ReadHeaderTimeout: 10 * time.Second}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	emit(Event{Event: "start", Time: now(), URLs: urls})
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = srv.Shutdown(sctx)
	emit(Event{Event: "stop", Time: now()})
	return err
}
