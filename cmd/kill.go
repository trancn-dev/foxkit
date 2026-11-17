package cmd

import (
	"fmt"
	"io"
	"strings"

	"foxkit/internal/kill"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "kill <pid|name>...",
		Short: "Kill process theo PID hoặc tên",
		RunE: func(c *cobra.Command, args []string) error {
			tree, _ := c.Flags().GetBool("tree")
			if len(args) == 0 {
				return output.Usage("KILL_NO_TARGET", "Thiếu PID hoặc tên process", "Ví dụ: foxkit kill node")
			}
			targets, notFound, err := kill.Resolve(c.Context(), args)
			if err != nil {
				return output.Fail("TASKLIST_FAILED", "Không đọc được danh sách process", `Thử chạy "tasklist" trong terminal`)
			}
			res := kill.Kill(c.Context(), targets, tree)
			res.Failed = append(notFound, res.Failed...)
			perr := printer(c).Success(res, func(w io.Writer) error {
				if len(res.Killed) > 0 {
					var s []string
					for _, t := range res.Killed {
						s = append(s, fmt.Sprintf("%s (%d)", t.Name, t.PID))
					}
					fmt.Fprintf(w, "Đã kill: %s\n", strings.Join(s, ", "))
				}
				if len(res.Failed) > 0 {
					var s []string
					for _, f := range res.Failed {
						s = append(s, fmt.Sprintf("%s (%d) — %s", f.Name, f.PID, f.Message))
					}
					fmt.Fprintf(w, "Thất bại: %s\n", strings.Join(s, "; "))
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			switch kill.ExitCode(res) {
			case "NEED_ADMIN":
				return output.NeedAdmin("Không đủ quyền kill một số process")
			case "KILL_PARTIAL":
				return output.Fail("KILL_PARTIAL", fmt.Sprintf("%d process không kill được", len(res.Failed)), "Xem danh sách bằng foxkit ps")
			}
			return nil
		},
	}
	c.Flags().Bool("tree", false, "Kill cả process con")
	rootCmd.AddCommand(c)
}
