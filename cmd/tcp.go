package cmd

import (
	"fmt"
	"io"
	"time"

	"foxkit/internal/output"
	"foxkit/internal/tcpcheck"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "tcp <host:port>...",
		Short: "Kiểm tra host:port có mở không",
		RunE: func(c *cobra.Command, targets []string) error {
			timeoutStr, _ := c.Flags().GetString("timeout")
			if len(targets) == 0 {
				return output.Usage("TCP_NO_TARGET", "Thiếu host:port", "Ví dụ: foxkit tcp localhost:3306")
			}
			if len(targets) > 100 {
				return output.Usage("ARGS_INVALID", "Tối đa 100 đích", "")
			}
			timeout, err := time.ParseDuration(timeoutStr)
			if err != nil || timeout <= 0 {
				return output.Usage("ARGS_INVALID", "--timeout không hợp lệ: "+timeoutStr, "Ví dụ: --timeout 3s")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			items := tcpcheck.CheckAll(ctx, targets, timeout)
			closed := 0
			for _, it := range items {
				if !it.Open {
					closed++
				}
			}
			perr := printer(c).Success(map[string]any{"items": items}, func(w io.Writer) error {
				for _, it := range items {
					label := it.Addr
					if label == "" {
						label = it.Target
					}
					if it.Open {
						fmt.Fprintf(w, "✓  %-24s %-15s %d ms\n", label, it.IP, it.Ms)
					} else {
						fmt.Fprintf(w, "✗  %-24s %-15s %s\n", label, it.IP, it.Error)
					}
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			if closed > 0 {
				return output.Fail("TCP_SOME_CLOSED", fmt.Sprintf("%d/%d đích không mở", closed, len(items)), "Timeout thường do firewall; từ chối thường do service chưa chạy")
			}
			return nil
		},
	}
	c.Flags().String("timeout", "3s", "Timeout mỗi đích")
	rootCmd.AddCommand(c)
}
