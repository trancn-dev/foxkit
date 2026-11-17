package cmd

import (
	"fmt"
	"io"
	"time"
	_ "time/tzdata" // nhúng dữ liệu múi giờ để chạy được trên máy không cài Go

	"foxkit/internal/output"
	"foxkit/internal/ts"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "ts [value]",
		Short: "Unix timestamp ↔ ngày giờ, nhiều múi giờ",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			tz, _ := c.Flags().GetString("tz")
			loc, err := time.LoadLocation(tz)
			if err != nil {
				return output.Usage("TS_TZ_INVALID", "Múi giờ không hợp lệ: "+tz, "Dùng tên IANA, ví dụ Asia/Ho_Chi_Minh")
			}
			value := ""
			if len(args) == 1 {
				value = args[0]
			}
			now := time.Now()
			t, kind, err := ts.Parse(value, loc, now)
			if err != nil {
				return output.Usage("TS_PARSE_FAILED", "Không nhận diện được: "+value, "Dạng hỗ trợ: 1763287727, 2026-11-16 19:08:47, 2026-11-16T19:08:47Z")
			}
			info := ts.Describe(t, loc, now)
			info.Detected = kind
			return printer(c).Success(info, func(w io.Writer) error {
				fmt.Fprintf(w, "Unix      %d   (ms: %d)\n", info.Unix, info.UnixMs)
				fmt.Fprintf(w, "Giờ       %s  %s  %s\n", info.Local, info.Weekday, info.Zone)
				fmt.Fprintf(w, "ISO       %s\n", info.ISO)
				fmt.Fprintf(w, "UTC       %s\n", info.UTC)
				_, err := fmt.Fprintf(w, "Cách đây  %s\n", info.Relative)
				return err
			})
		},
	}
	c.Flags().String("tz", "Local", "Múi giờ IANA, ví dụ Asia/Ho_Chi_Minh")
	rootCmd.AddCommand(c)
}
