package cmd

import (
	"fmt"
	"io"

	"foxkit/internal/netinfo"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "ip",
		Short: "IP LAN và IP public",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			public, _ := c.Flags().GetBool("public")
			all, _ := c.Flags().GetBool("all")
			v6, _ := c.Flags().GetBool("v6")
			lan, err := netinfo.LANAddrs(all, v6)
			if err != nil {
				return output.Fail("IP_LIST_FAILED", "Không liệt kê được card mạng: "+err.Error(), "")
			}
			res := netinfo.Result{LAN: lan}
			res.Primary, _ = netinfo.PrimaryIP() // không có route ra ngoài (offline) thì để trống, không phải lỗi
			var pubErr error
			if public {
				res.Public, pubErr = netinfo.PublicIP(c.Context())
			}
			perr := printer(c).Success(res, func(w io.Writer) error {
				fmt.Fprintln(w, "LAN")
				for _, a := range lan {
					mark := ""
					if a.IP == res.Primary {
						mark = "   ← dùng để ra Internet"
					}
					fmt.Fprintf(w, "  %-14s %s/%d%s\n", a.Iface, a.IP, a.Prefix, mark)
				}
				if res.Public != "" {
					fmt.Fprintf(w, "Public         %s\n", res.Public)
				}
				if res.Primary != "" {
					fmt.Fprintf(w, "Mở trên điện thoại: http://%s:<port>\n", res.Primary)
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			if pubErr != nil {
				return output.Fail("IP_PUBLIC_FAILED", "Không lấy được IP public", "Kiểm tra kết nối Internet hoặc proxy")
			}
			return nil
		},
	}
	c.Flags().Bool("public", false, "Lấy IP public (gửi request ra Internet)")
	c.Flags().Bool("all", false, "Gồm card ảo và card đang down")
	c.Flags().Bool("v6", false, "Gồm IPv6")
	rootCmd.AddCommand(c)
}
