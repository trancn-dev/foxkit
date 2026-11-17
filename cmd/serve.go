package cmd

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/serve"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "serve [dir]",
		Short: "Chạy web server tĩnh cho thư mục, hỗ trợ SPA",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			port, _ := f.GetInt("port")
			host, _ := f.GetString("host")
			opt := serve.Options{Dir: dir, Host: host, Port: port}
			opt.SPA, _ = f.GetBool("spa")
			opt.CORS, _ = f.GetBool("cors")
			opt.NoCache, _ = f.GetBool("no-cache")
			opt.List, _ = f.GetBool("list")
			open, _ := f.GetBool("open")
			if port < 0 || port > 65535 {
				return output.PortInvalid()
			}
			if !fsutil.IsDir(dir) {
				return output.DirNotFound(dir)
			}
			abs, err := filepath.Abs(dir)
			if err != nil {
				return output.Internal(err)
			}
			opt.Dir = abs
			if opt.SPA && !fsutil.Exists(filepath.Join(abs, "index.html")) {
				return output.Usage("SERVE_SPA_NO_INDEX", "--spa cần có index.html trong "+abs, "Thư mục build Vue thường là dist/")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			p := printer(c)
			var mu sync.Mutex // request đến từ nhiều goroutine, tránh in chồng dòng
			emit := func(e serve.Event) {
				mu.Lock()
				defer mu.Unlock()
				if p.JSON {
					_ = p.Event(e) // stdout đóng (pipe bị ngắt) thì không còn gì để báo
					return
				}
				w := c.OutOrStdout()
				switch e.Event {
				case "start":
					mode := ""
					if opt.SPA {
						mode = "  (SPA)"
					}
					fmt.Fprintf(w, "Đang phục vụ %s%s\n", abs, mode)
					for _, u := range e.URLs {
						fmt.Fprintf(w, "  %s\n", u)
					}
					if len(e.URLs) > 1 {
						fmt.Fprintln(w, "Nếu điện thoại không vào được, cho phép foxkit trong Windows Firewall.")
					}
					fmt.Fprintln(w, "Nhấn Ctrl+C để dừng.")
					if open {
						_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", e.URLs[0]).Start() // không mở được trình duyệt thì thôi, server vẫn chạy
					}
				case "request":
					t, _ := time.Parse(time.RFC3339, e.Time)
					fmt.Fprintf(w, "%s  %d  %-4s %-30s %9s  %dms\n", t.Format("15:04:05"), e.Status, e.Method, e.Path, fsutil.FormatBytes(e.Bytes), e.Ms)
				case "stop":
					fmt.Fprintln(w, "Đã dừng server.")
				}
			}
			if err := serve.Run(ctx, opt, emit); err != nil {
				if errors.Is(err, serve.ErrNoPort) {
					return output.Fail("SERVE_NO_PORT", "Không còn port trống từ "+fmt.Sprint(port), "Chỉ định --port khác hoặc xem foxkit port --list")
				}
				return output.Fail("SERVE_FAILED", err.Error(), "")
			}
			return nil
		},
	}
	f := c.Flags()
	f.Int("port", 8080, "Port (0 = hệ điều hành chọn)")
	f.String("host", "0.0.0.0", "Host lắng nghe")
	f.Bool("spa", false, "Chế độ SPA")
	f.Bool("cors", false, "Bật CORS")
	f.Bool("no-cache", true, "Tắt cache")
	f.Bool("open", false, "Mở trình duyệt")
	f.Bool("list", false, "Cho liệt kê thư mục")
	rootCmd.AddCommand(c)
}
