package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"foxkit/internal/fsutil"
	"foxkit/internal/git"
	"foxkit/internal/output"
	"foxkit/internal/repos"

	"github.com/spf13/cobra"
)

func summary(r repos.Repo) string {
	if r.Error != "" && r.Branch == "" {
		return "⚠ " + r.Error
	}
	var p []string
	if r.Conflicts > 0 {
		p = append(p, fmt.Sprintf("✗ %d xung đột", r.Conflicts))
	}
	if r.Staged > 0 {
		p = append(p, fmt.Sprintf("● %d staged", r.Staged))
	}
	if r.Modified > 0 {
		p = append(p, fmt.Sprintf("✎ %d sửa", r.Modified))
	}
	if r.Untracked > 0 {
		p = append(p, fmt.Sprintf("+ %d mới", r.Untracked))
	}
	if r.Ahead > 0 {
		p = append(p, fmt.Sprintf("↑%d chưa push", r.Ahead))
	}
	if r.Behind > 0 {
		p = append(p, fmt.Sprintf("↓%d tụt so với %s", r.Behind, r.Upstream))
	}
	if r.Upstream == "" {
		p = append(p, "⚠ chưa có upstream")
	}
	if r.Stashes > 0 {
		p = append(p, fmt.Sprintf("%d stash", r.Stashes))
	}
	if r.Error != "" {
		p = append(p, "⚠ "+r.Error)
	}
	if len(p) == 0 {
		return "✓ sạch"
	}
	return strings.Join(p, " · ")
}

func init() {
	c := &cobra.Command{
		Use:   "repos <dir>",
		Short: "Repo nào chưa commit, chưa push, bị tụt",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			depth, _ := c.Flags().GetInt("depth")
			dirty, _ := c.Flags().GetBool("dirty")
			fetch, _ := c.Flags().GetBool("fetch")
			if !fsutil.IsDir(args[0]) {
				return output.DirNotFound(args[0])
			}
			if depth < 1 || depth > 6 {
				return output.Usage("ARGS_INVALID", "--depth phải từ 1 đến 6", "")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			all, err := repos.Scan(ctx, args[0], depth, fetch)
			if errors.Is(err, git.ErrNotFound) {
				return output.GitNotFound()
			}
			if err != nil {
				return canceled(err)
			}
			attention := 0
			items := make([]repos.Repo, 0, len(all))
			for _, r := range all {
				if r.Attention() {
					attention++
				}
				if !dirty || r.Attention() {
					items = append(items, r)
				}
			}
			return printer(c).Success(map[string]any{"items": items, "attention": attention}, func(w io.Writer) error {
				fmt.Fprintf(w, "%-18s %-20s %s\n", "REPO", "BRANCH", "TRẠNG THÁI")
				for _, r := range items {
					fmt.Fprintf(w, "%-18s %-20s %s\n", r.Name, r.Branch, summary(r))
				}
				_, err := fmt.Fprintf(w, "%d repo · %d cần chú ý\n", len(all), attention)
				return err
			})
		},
	}
	c.Flags().Int("depth", 3, "Độ sâu tìm .git (1-6)")
	c.Flags().Bool("dirty", false, "Chỉ repo cần chú ý")
	c.Flags().Bool("fetch", false, "Fetch trước (chậm, cần mạng)")
	rootCmd.AddCommand(c)
}
