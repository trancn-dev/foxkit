package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/todo"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "todo [dir]",
		Short: "Tìm TODO/FIXME/HACK trong mã nguồn",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			tags, _ := c.Flags().GetStringSlice("tags")
			exts, _ := c.Flags().GetStringSlice("ext")
			author, _ := c.Flags().GetBool("author")
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if !fsutil.IsDir(dir) {
				return output.DirNotFound(dir)
			}
			if !todo.ValidTags(tags) {
				return output.Usage("TAGS_INVALID", "Tag rỗng hoặc có ký tự lạ", "Ví dụ: --tags TODO,FIXME")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			rep, err := todo.Run(ctx, todo.Options{Dir: dir, Tags: tags, Exts: exts, Author: author})
			if err != nil {
				return canceled(err)
			}
			return printer(c).Success(rep, func(w io.Writer) error {
				last := ""
				for _, it := range rep.Items {
					if it.File != last {
						fmt.Fprintln(w, it.File)
						last = it.File
					}
					who := ""
					if it.Author != "" {
						who = "  (" + it.Author + ")"
					}
					fmt.Fprintf(w, "  %-4d %-6s %s%s\n", it.Line, it.Tag, it.Text, who)
				}
				keys := make([]string, 0, len(rep.Counts))
				for k := range rep.Counts {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				var parts []string
				for _, k := range keys {
					parts = append(parts, fmt.Sprintf("%s %d", k, rep.Counts[k]))
				}
				if len(parts) == 0 {
					parts = []string{"không có"}
				}
				_, err := fmt.Fprintf(w, "\nTổng: %s (trong %s file)\n", strings.Join(parts, " · "), fsutil.FormatInt(rep.Files))
				return err
			})
		},
	}
	c.Flags().StringSlice("tags", todo.DefaultTags, "Tag cần tìm")
	c.Flags().StringSlice("ext", nil, "Đuôi file")
	c.Flags().Bool("author", false, "Kèm tác giả (git blame)")
	rootCmd.AddCommand(c)
}
