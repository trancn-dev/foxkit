package cmd

import (
	"fmt"
	"io"
	"slices"
	"time"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/prune"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "prune <dir>",
		Short: "Xóa node_modules, vendor của dự án lâu không dùng",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			targets, _ := c.Flags().GetStringSlice("target")
			olderStr, _ := c.Flags().GetString("older")
			depth, _ := c.Flags().GetInt("depth")
			apply, _ := c.Flags().GetBool("apply")
			if !fsutil.IsDir(args[0]) {
				return output.DirNotFound(args[0])
			}
			for _, t := range targets {
				if !slices.Contains(prune.SafeTargets, t) {
					return output.Usage("PRUNE_TARGET_INVALID", "Thư mục không được phép xóa: "+t, "Cho phép: node_modules, vendor, .next, .nuxt, dist, .turbo, .cache")
				}
			}
			var older time.Duration
			if olderStr != "0" {
				d, err := fsutil.ParseAge(olderStr)
				if err != nil {
					return output.AgeInvalid(olderStr)
				}
				older = d
			}
			if depth < 1 || depth > 10 {
				return output.Usage("ARGS_INVALID", "--depth phải từ 1 đến 10", "")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			rep, err := prune.Run(ctx, prune.Options{Dir: args[0], Targets: targets, Older: older, Depth: depth, Apply: apply})
			if err != nil {
				return canceled(err)
			}
			perr := printer(c).Success(rep, func(w io.Writer) error {
				projects := map[string]bool{}
				for _, it := range rep.Items {
					projects[it.Project] = true
				}
				fmt.Fprintf(w, "Tìm thấy %d thư mục trong %d dự án không đụng tới hơn %s (tổng %s):\n\n", len(rep.Items), len(projects), olderStr, fsutil.FormatBytes(rep.TotalBytes))
				for _, it := range rep.Items {
					state := ""
					switch {
					case it.Deleted:
						state = "  [đã xóa]"
					case it.Error != "":
						state = "  [LỖI: " + it.Error + "]"
					}
					fmt.Fprintf(w, "  %9s  %-12s  %s  (sửa lần cuối %s)%s\n", fsutil.FormatBytes(it.Bytes), it.Target, it.Project, it.LastActivity.Format("2006-01-02"), state)
				}
				fmt.Fprintf(w, "Bỏ qua %d dự án còn hoạt động.\n", rep.Skipped)
				if apply {
					fmt.Fprintf(w, "Đã giải phóng %s.\n", fsutil.FormatBytes(rep.FreedBytes))
				} else {
					fmt.Fprintln(w, output.DryRunNote)
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			for _, it := range rep.Items {
				if it.Error != "" {
					return output.Fail("PRUNE_PARTIAL", "Có thư mục không xóa được (xem report)", "Tắt dev server/IDE đang mở dự án rồi chạy lại; hoặc dùng foxkit lock <path>")
				}
			}
			return nil
		},
	}
	c.Flags().StringSlice("target", []string{"node_modules", "vendor"}, "Loại thư mục (node_modules, vendor, .next, .nuxt, dist, .turbo, .cache)")
	c.Flags().String("older", "30d", "Dự án không đụng tới hơn mốc này (0 = mọi dự án)")
	c.Flags().Int("depth", 4, "Độ sâu tối đa (1-10)")
	c.Flags().Bool("apply", false, "Xóa thật")
	rootCmd.AddCommand(c)
}
