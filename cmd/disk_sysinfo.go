package cmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"foxkit/internal/disk"
	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/sysinfo"

	"github.com/spf13/cobra"
)

func bar(pct float64) string {
	n := int(pct/100*20 + 0.5)
	if n > 20 {
		n = 20
	}
	return strings.Repeat("█", n) + strings.Repeat("░", 20-n)
}

func init() {
	d := &cobra.Command{
		Use:   "disk",
		Short: "Dung lượng các ổ, cảnh báo ổ sắp đầy",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			all, _ := c.Flags().GetBool("all")
			warn, _ := c.Flags().GetInt("warn")
			if warn < 1 || warn > 100 {
				return output.Usage("WARN_INVALID", "--warn phải từ 1 đến 100", "Ví dụ: --warn 90")
			}
			drives, err := disk.List(all, float64(warn))
			if err != nil {
				return output.Fail("DISK_LIST_FAILED", "Không liệt kê được ổ đĩa: "+err.Error(), "")
			}
			return printer(c).Success(map[string]any{"items": drives}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "Ổ\tLOẠI\tNHÃN\tFS\tDUNG LƯỢNG\tTRỐNG\tĐÃ DÙNG")
				var full []string
				for _, d := range drives {
					if d.Error != "" {
						fmt.Fprintf(tw, "%s\t%s\t\t\t\t\t(%s)\n", d.Root, d.Type, d.Error)
						continue
					}
					mark := ""
					if d.Warn {
						mark = "  ⚠"
						full = append(full, d.Root)
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%.0f%%  %s%s\n", d.Root, d.Type, d.Label, d.FS, fsutil.FormatBytes(int64(d.TotalBytes)), fsutil.FormatBytes(int64(d.FreeBytes)), d.UsedPct, bar(d.UsedPct), mark)
				}
				if err := tw.Flush(); err != nil {
					return err
				}
				for _, r := range full {
					fmt.Fprintf(w, "Ổ %s sắp đầy. Thử: foxkit temp --apply, foxkit prune <thư mục dự án>, foxkit size %sUsers\n", r, r)
				}
				return nil
			})
		},
	}
	d.Flags().Bool("all", false, "Gồm ổ không sẵn sàng")
	d.Flags().Int("warn", 90, "Ngưỡng cảnh báo % đã dùng")
	rootCmd.AddCommand(d)

	s := &cobra.Command{
		Use:   "sysinfo",
		Short: "Windows, CPU, RAM, uptime",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			in, err := sysinfo.Get()
			if err != nil {
				return output.Fail("SYSINFO_FAILED", "Không đọc được thông tin máy: "+err.Error(), "")
			}
			return printer(c).Success(in, func(w io.Writer) error {
				admin := "không phải admin"
				if in.IsAdmin {
					admin = "đang chạy admin"
				}
				fmt.Fprintf(w, "Windows     %s %s (build %s, %s)\n", in.OS, in.DisplayVer, in.Build, in.Arch)
				fmt.Fprintf(w, "Máy         %s · user %s · %s\n", in.Hostname, in.User, admin)
				fmt.Fprintf(w, "CPU         %s · %d luồng\n", in.CPU, in.Cores)
				fmt.Fprintf(w, "RAM         %s · trống %s · đã dùng %d%%\n", fsutil.FormatBytes(int64(in.RAMBytes)), fsutil.FormatBytes(int64(in.RAMFreeBytes)), in.RAMUsedPct)
				fmt.Fprintf(w, "Bật từ      %s (%s)\n", in.BootTime, in.Uptime)
				for _, wn := range in.Warnings {
					fmt.Fprintf(c.ErrOrStderr(), "Cảnh báo: %s\n", wn)
				}
				return nil
			})
		},
	}
	rootCmd.AddCommand(s)
}
