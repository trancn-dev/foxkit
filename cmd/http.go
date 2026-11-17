package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"foxkit/internal/fsutil"
	"foxkit/internal/httpcheck"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "http <url>",
		Short: "Status, thời gian, redirect, SSL của một URL",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			method, _ := f.GetString("method")
			headers, _ := f.GetStringArray("header")
			data, _ := f.GetString("data")
			timeoutStr, _ := f.GetString("timeout")
			noFollow, _ := f.GetBool("no-follow")
			insecure, _ := f.GetBool("insecure")
			body, _ := f.GetBool("body")

			u, err := httpcheck.NormalizeURL(args[0])
			if err != nil {
				return output.Usage("URL_INVALID", err.Error(), "Ví dụ: foxkit http https://example.com")
			}
			method = strings.ToUpper(method)
			if !slices.Contains([]string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, method) {
				return output.Usage("ARGS_INVALID", "--method không hợp lệ: "+method, "Chọn: GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
			}
			for _, h := range headers {
				if _, _, err := httpcheck.ParseHeader(h); err != nil {
					return output.Usage("HEADER_INVALID", err.Error(), `Dạng "Key: Value"`)
				}
			}
			timeout, err := time.ParseDuration(timeoutStr)
			if err != nil || timeout <= 0 {
				return output.Usage("ARGS_INVALID", "--timeout không hợp lệ: "+timeoutStr, "Ví dụ: --timeout 10s")
			}
			if strings.HasPrefix(data, "@") {
				b, err := os.ReadFile(data[1:])
				if err != nil {
					return output.FileNotFound(data[1:])
				}
				data = string(b)
			}
			hasCT := false
			for _, h := range headers {
				if k, _, _ := httpcheck.ParseHeader(h); strings.EqualFold(k, "content-type") {
					hasCT = true
				}
			}
			if data != "" && !hasCT && json.Valid([]byte(data)) {
				headers = append(headers, "Content-Type: application/json")
			}
			res, err := httpcheck.Check(c.Context(), httpcheck.Options{
				URL: u, Method: method, Headers: headers, Data: data, Timeout: timeout,
				NoFollow: noFollow, Insecure: insecure, WantBody: body, UserAgent: "foxkit/" + currentVersion().Version,
			})
			if err != nil {
				var he *httpcheck.Error
				if errors.As(err, &he) {
					switch he.Kind {
					case "timeout":
						return output.Fail("HTTP_TIMEOUT", "Quá thời gian chờ "+timeoutStr, "Tăng --timeout hoặc kiểm tra server")
					case "tls":
						return output.Fail("HTTP_TLS_FAILED", "Chứng chỉ không hợp lệ: "+he.Error(), "Dev local self-signed thì thêm --insecure")
					case "redirects":
						return output.Fail("TOO_MANY_REDIRECTS", "Quá 10 lần chuyển hướng", "Thử --no-follow để xem từng bước")
					}
				}
				return output.Fail("HTTP_CONNECT_FAILED", "Không kết nối được: "+err.Error(), "Kiểm tra domain bằng foxkit dns <host>")
			}
			return printer(c).Success(res, func(w io.Writer) error {
				fmt.Fprintf(w, "%s %s → %d  (%s, %s)\n\n", method, res.FinalURL, res.Status, res.Proto, fsutil.FormatBytes(res.Bytes))
				for _, h := range res.Redirects {
					fmt.Fprintf(w, "  Redirect   %s → %d\n", h.URL, h.Status)
				}
				t := res.Timing
				fmt.Fprintf(w, "  DNS        %d ms\n  Kết nối    %d ms\n", t.DNSMs, t.ConnectMs)
				if res.TLS != nil {
					fmt.Fprintf(w, "  TLS        %d ms\n", t.TLSMs)
				}
				fmt.Fprintf(w, "  Chờ        %d ms\n  Tổng       %d ms\n", t.TTFBMs, t.TotalMs)
				if res.TLS != nil {
					fmt.Fprintf(w, "\n  SSL        %s · hết hạn %s (còn %d ngày) · %s\n", res.TLS.Issuer, res.TLS.Expires.Format("2006-01-02"), res.TLS.DaysLeft, res.TLS.Version)
					if res.TLS.DaysLeft < 14 {
						fmt.Fprintln(w, "  ⚠ Chứng chỉ sắp hết hạn (dưới 14 ngày)")
					}
				}
				keys := make([]string, 0, len(res.Headers))
				for k := range res.Headers {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for i, k := range keys {
					lead := "  Header     "
					if i > 0 {
						lead = "             "
					}
					fmt.Fprintf(w, "%s%s: %s\n", lead, k, res.Headers[k])
				}
				if res.Body != "" {
					fmt.Fprintf(w, "\n%s\n", res.Body)
				}
				return nil
			})
		},
	}
	f := c.Flags()
	f.String("method", "GET", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
	f.StringArray("header", nil, `Header "Key: Value" (lặp nhiều lần)`)
	f.String("data", "", "Body; bắt đầu bằng @ thì đọc từ file")
	f.String("timeout", "10s", "Timeout")
	f.Bool("no-follow", false, "Không theo redirect")
	f.Bool("insecure", false, "Bỏ kiểm tra chứng chỉ SSL")
	f.Bool("body", false, "In body (tối đa 64KB)")
	rootCmd.AddCommand(c)
}
