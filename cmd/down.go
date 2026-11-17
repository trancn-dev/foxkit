package cmd

import (
	"errors"
	"fmt"
	"io"
	"time"

	"foxkit/internal/down"
	"foxkit/internal/fsutil"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func downError(err error, retries int) error {
	var e *down.Error
	if !errors.As(err, &e) {
		return canceled(err)
	}
	switch e.Kind {
	case "url":
		return output.Usage("URL_INVALID", e.Error(), "Ví dụ: foxkit down https://example.com/file.zip")
	case "exists":
		return output.Fail("FILE_EXISTS", e.Error(), "Thêm --force để ghi đè")
	case "status":
		return output.Fail("HTTP_STATUS", fmt.Sprintf("Server trả %d", e.Status), fmt.Sprintf("Server trả %d; kiểm tra lại URL", e.Status))
	case "checksum":
		return output.Fail("DOWN_CHECKSUM", e.Error(), "File có thể tải lỗi hoặc bị thay đổi; đã xóa file, hãy tải lại")
	case "timeout":
		return output.Fail("HTTP_TIMEOUT", "Quá thời gian chờ", "Kiểm tra mạng, thử lại với --retries")
	case "disk":
		return output.Fail("DISK_FULL", "Hết chỗ trống trên ổ đĩa", "Giải phóng dung lượng: foxkit size, foxkit temp")
	}
	return output.Fail("HTTP_CONNECT_FAILED", "Không kết nối được: "+e.Error(), "Kiểm tra mạng, thử lại với --retries")
}

func init() {
	c := &cobra.Command{
		Use:   "down <url>",
		Short: "Tải file từ URL, tiếp tục khi đứt, kiểm tra checksum",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			out, _ := f.GetString("out")
			sha, _ := f.GetString("sha256")
			headers, _ := f.GetStringArray("header")
			resume, _ := f.GetBool("resume")
			force, _ := f.GetBool("force")
			retries, _ := f.GetInt("retries")
			timeout, _ := f.GetDuration("timeout")
			if retries < 0 || retries > 10 {
				return output.Usage("ARGS_INVALID", "--retries phải từ 0 đến 10", "")
			}
			if timeout <= 0 {
				return output.Usage("ARGS_INVALID", "--timeout phải lớn hơn 0", "Ví dụ: --timeout 30s")
			}
			if sha != "" && len(sha) != 64 {
				return output.Usage("ARGS_INVALID", "--sha256 phải là 64 ký tự hex", "")
			}
			h, err := down.ParseHeaders(headers)
			if err != nil {
				return output.Usage("HEADER_INVALID", err.Error(), `Dạng "Key: Value"`)
			}
			ctx, stop := cmdContext(c)
			defer stop()
			p := printer(c)
			tty := !p.JSON && isTerminal(c.ErrOrStderr())
			emit := func(e down.Event) {
				switch {
				case p.JSON:
					_ = p.Event(e) // pipe đóng thì lần tải cũng sắp kết thúc
				case tty && (e.Event == "progress" || e.Event == "start"):
					bar := down.Bar(e.Bytes, e.Total, 20)
					eta := down.ETA(e.Bytes, e.Total, e.BytesPerSec)
					if eta != "" {
						eta = "  còn " + eta
					}
					fmt.Fprintf(c.ErrOrStderr(), "\r%s  %s / %s  %s/s%s   ", bar, fsutil.FormatBytes(e.Bytes), fsutil.FormatBytes(e.Total), fsutil.FormatBytes(e.BytesPerSec), eta)
				case e.Event == "retry" && !p.JSON:
					fmt.Fprintf(c.ErrOrStderr(), "\nLỗi mạng, thử lại lần %d...\n", e.Attempt)
				}
			}
			res, err := down.Download(ctx, down.Options{URL: args[0], Out: out, Headers: h, Resume: resume, Force: force, Retries: retries, Timeout: timeout, SHA256: sha, UserAgent: "foxkit/" + currentVersion().Version}, emit)
			if tty {
				fmt.Fprintln(c.ErrOrStderr())
			}
			if err != nil {
				return downError(err, retries)
			}
			if p.JSON {
				return nil // sự kiện "done" đã in
			}
			return p.Success(res, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Đã tải %s (%s) trong %s → %s  sha256 %.12s…\n", baseName(res.Path), fsutil.FormatBytes(res.Bytes), (time.Duration(res.DurationMs) * time.Millisecond).Round(time.Second), res.Path, res.SHA256)
				return err
			})
		},
	}
	f := c.Flags()
	f.String("out", "", "Thư mục hoặc đường dẫn file đích")
	f.String("sha256", "", "SHA-256 mong đợi")
	f.StringArray("header", nil, `Header "Key: Value" (lặp nhiều lần)`)
	f.Bool("resume", true, "Tiếp tục nếu dở dang")
	f.Bool("force", false, "Ghi đè file đã có")
	f.Int("retries", 3, "Số lần thử lại (0-10)")
	f.Duration("timeout", 30*time.Second, "Timeout kết nối và chờ header")
	rootCmd.AddCommand(c)
}

func baseName(p string) string { // filepath.Base cũng được; viết tay để cmd không cần thêm import
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '\\' || p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}
