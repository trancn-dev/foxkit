package cmd

import (
	"fmt"
	"io"
	"net"
	"slices"
	"strings"

	"foxkit/internal/dns"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "dns [host]",
		Short: "Tra bản ghi DNS, xóa cache DNS",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			flush, _ := c.Flags().GetBool("flush")
			types, _ := c.Flags().GetStringSlice("type")
			server, _ := c.Flags().GetString("server")
			p := printer(c)
			if flush {
				if len(args) > 0 {
					return output.Usage("ARGS_INVALID", "--flush không dùng chung với host", "Chạy: foxkit dns --flush")
				}
				if err := dns.Flush(c.Context()); err != nil {
					return output.Fail("DNS_FLUSH_FAILED", "Không xóa được cache DNS", `Thử chạy "ipconfig /flushdns" bằng quyền Administrator`)
				}
				return p.Success(map[string]bool{"flushed": true}, func(w io.Writer) error {
					_, err := fmt.Fprintln(w, "Đã xóa cache DNS.")
					return err
				})
			}
			if len(args) == 0 {
				return output.Usage("DNS_NO_HOST", "Thiếu domain cần tra", "Ví dụ: foxkit dns example.com")
			}
			if server != "" && net.ParseIP(server) == nil {
				return output.Usage("DNS_SERVER_INVALID", "DNS server phải là địa chỉ IP: "+server, "Ví dụ: --server 1.1.1.1")
			}
			var want []string
			for _, t := range types {
				t = strings.ToUpper(strings.TrimSpace(t))
				if t == "ALL" {
					want = dns.AllTypes
					break
				}
				if !slices.Contains(dns.AllTypes, t) {
					return output.Usage("DNS_TYPE_INVALID", "Loại bản ghi không hợp lệ: "+t, "Chọn: A, AAAA, CNAME, MX, TXT, NS, all")
				}
				if !slices.Contains(want, t) {
					want = append(want, t)
				}
			}
			host := dns.NormalizeHost(args[0])
			res := dns.Lookup(c.Context(), host, want, server)
			perr := p.Success(res, func(w io.Writer) error {
				fmt.Fprintf(w, "%s  (qua DNS %s, %dms)\n", res.Host, res.Server, res.Ms)
				rec := res.Records
				for _, t := range want {
					var vals []string
					switch t {
					case "A":
						vals = rec.A
					case "AAAA":
						vals = rec.AAAA
					case "CNAME":
						if rec.CNAME != "" {
							vals = []string{rec.CNAME}
						}
					case "MX":
						for _, m := range rec.MX {
							vals = append(vals, fmt.Sprintf("%d %s", m.Pref, m.Host))
						}
					case "TXT":
						vals = rec.TXT
					case "NS":
						vals = rec.NS
					}
					if len(vals) == 0 {
						fmt.Fprintf(w, "  %-6s (%s)\n", t, res.Errors[t])
						continue
					}
					for _, v := range vals {
						fmt.Fprintf(w, "  %-6s %s\n", t, v)
					}
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			if len(res.Errors) == len(want) {
				return output.Fail("DNS_NOT_FOUND", "Không tra được bản ghi nào cho "+host, "Kiểm tra chính tả domain")
			}
			return nil
		},
	}
	c.Flags().StringSlice("type", []string{"A", "AAAA", "CNAME"}, "A,AAAA,CNAME,MX,TXT,NS hoặc all")
	c.Flags().String("server", "", "Tra qua DNS server chỉ định")
	c.Flags().Bool("flush", false, "Xóa cache DNS")
	rootCmd.AddCommand(c)
}
