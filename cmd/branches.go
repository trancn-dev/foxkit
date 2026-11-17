package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"foxkit/internal/branches"
	"foxkit/internal/fsutil"
	"foxkit/internal/git"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	parent := &cobra.Command{Use: "branches", Short: "Quản lý branch git"}
	clean := &cobra.Command{
		Use:   "clean [dir]",
		Short: "Xóa branch local đã merge hoặc đã mất trên remote",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			base, _ := c.Flags().GetString("base")
			gone, _ := c.Flags().GetBool("gone")
			keep, _ := c.Flags().GetStringSlice("keep")
			apply, _ := c.Flags().GetBool("apply")
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if !fsutil.IsDir(dir) {
				return output.DirNotFound(dir)
			}
			ctx, stop := cmdContext(c)
			defer stop()
			plan, err := branches.Build(ctx, dir, base, gone, keep)
			switch {
			case errors.Is(err, git.ErrNotFound):
				return output.GitNotFound()
			case errors.Is(err, branches.ErrNotRepo):
				return output.Usage("GIT_NOT_REPO", "Thư mục này không phải repo git: "+dir, "Chạy trong thư mục repo hoặc truyền đường dẫn")
			case errors.Is(err, branches.ErrBaseNotFound):
				return output.Usage("BRANCH_BASE_NOT_FOUND", "Không đoán được branch gốc (main/master/develop)", "Chỉ định --base")
			case err != nil:
				return canceled(err)
			}
			if apply && len(plan.Delete) > 0 {
				if err := branches.Apply(ctx, dir, &plan); err != nil {
					return canceled(err)
				}
			}
			perr := printer(c).Success(plan, func(w io.Writer) error {
				fmt.Fprintf(w, "Base: %s\n", plan.Base)
				for _, reason := range []struct{ key, title string }{{"merged", "merged"}, {"gone", "gone trên remote"}} {
					var rows []string
					for _, d := range plan.Delete {
						if d.Reason == reason.key {
							rows = append(rows, fmt.Sprintf("  %-28s %s", d.Name, d.Last))
						}
					}
					if len(rows) > 0 {
						verb := "Sẽ xóa"
						if plan.Applied {
							verb = "Đã xử lý"
						}
						fmt.Fprintf(w, "%s (%s):\n%s\n", verb, reason.title, strings.Join(rows, "\n"))
					}
				}
				if len(plan.Delete) == 0 {
					fmt.Fprintln(w, "Không có branch nào cần dọn.")
				}
				if len(plan.Failed) > 0 {
					fmt.Fprintf(w, "Không xóa được: %s\n", strings.Join(plan.Failed, ", "))
				}
				fmt.Fprintf(w, "Giữ: %s\n", strings.Join(plan.Kept, ", "))
				if !apply {
					fmt.Fprintln(w, output.DryRunNote)
					if gone {
						fmt.Fprintln(w, "Gợi ý: chạy `git fetch --prune` trước để biết branch nào đã mất trên remote.")
					}
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			if len(plan.Failed) > 0 {
				return output.Fail("BRANCHES_PARTIAL", fmt.Sprintf("%d branch không xóa được", len(plan.Failed)), "Xem mục failed")
			}
			return nil
		},
	}
	clean.Flags().String("base", "", "Branch gốc (mặc định tự đoán)")
	clean.Flags().Bool("gone", false, "Gồm branch có upstream đã mất trên remote")
	clean.Flags().StringSlice("keep", nil, "Branch luôn giữ")
	clean.Flags().Bool("apply", false, "Xóa thật")
	parent.AddCommand(clean)
	rootCmd.AddCommand(parent)
}
