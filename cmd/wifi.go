package cmd

import (
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	"foxkit/internal/output"
	"foxkit/internal/wifi"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "wifi [ssid]",
		Short: "Xem mật khẩu Wi-Fi đã lưu",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			ssid := ""
			if len(args) == 1 {
				ssid = args[0]
			}
			items, err := wifi.Export(c.Context(), ssid)
			if errors.Is(err, wifi.ErrUnavailable) {
				return output.Fail("WIFI_UNAVAILABLE", "Không đọc được Wi-Fi trên máy này", "Máy cần card Wi-Fi và dịch vụ WLAN AutoConfig đang chạy")
			}
			if err != nil {
				return output.Internal(err)
			}
			if ssid != "" && len(items) == 0 {
				return output.Fail("WIFI_NOT_FOUND", "Không có profile Wi-Fi tên "+ssid, "Chạy foxkit wifi để xem danh sách")
			}
			return printer(c).Success(map[string]any{"items": items}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "SSID\tBẢO MẬT\tMẬT KHẨU")
				for _, p := range items {
					pw := p.Password
					switch {
					case p.Error != "":
						pw = "(cần quyền admin)"
					case p.Auth == "open":
						pw = "(mạng mở)"
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", p.SSID, p.Auth, pw)
				}
				return tw.Flush()
			})
		},
	}
	rootCmd.AddCommand(c)
}
