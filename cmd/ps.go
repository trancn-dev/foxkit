package cmd

import (
	"fmt"
	"io"
	"slices"
	"text/tabwriter"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/ps"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "ps [filter]",
		Short: "Process đang chạy, sắp theo RAM, gộp theo tên",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			by, _ := c.Flags().GetString("sort")
			top, _ := c.Flags().GetInt("top")
			group, _ := c.Flags().GetBool("group")
			if !slices.Contains([]string{"mem", "name", "pid"}, by) {
				return output.Usage("SORT_INVALID", "Kiểu sắp xếp không hợp lệ: "+by, "Chọn: mem, name, pid")
			}
			if top < 0 {
				return output.Usage("ARGS_INVALID", "--top không được âm", "")
			}
			list, err := ps.List(c.Context())
			if err != nil {
				return output.Fail("TASKLIST_FAILED", "Không đọc được danh sách process", `Thử chạy "tasklist" trong terminal`)
			}
			if len(args) == 1 {
				list = ps.Filter(list, args[0])
			}
			var total int64
			for _, p := range list {
				total += p.MemBytes
			}
			p := printer(c)
			if group {
				gs := ps.GroupByName(list)
				ps.SortGroups(gs, by)
				n := len(gs)
				if top > 0 && len(gs) > top {
					gs = gs[:top]
				}
				return p.Success(map[string]any{"groups": gs, "total": n}, func(w io.Writer) error {
					tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
					fmt.Fprintln(tw, "NAME\tCOUNT\tMEM")
					for _, g := range gs {
						fmt.Fprintf(tw, "%s\t%d\t%s\n", g.Name, g.Count, fsutil.FormatBytes(g.MemBytes))
					}
					if err := tw.Flush(); err != nil {
						return err
					}
					_, err := fmt.Fprintf(w, "%d/%d nhóm · tổng %s\n", len(gs), n, fsutil.FormatBytes(total))
					return err
				})
			}
			ps.Sort(list, by)
			n := len(list)
			if top > 0 && len(list) > top {
				list = list[:top]
			}
			return p.Success(map[string]any{"items": list, "total": n}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "PID\tNAME\tMEM")
				for _, x := range list {
					fmt.Fprintf(tw, "%d\t%s\t%s\n", x.PID, x.Name, fsutil.FormatBytes(x.MemBytes))
				}
				if err := tw.Flush(); err != nil {
					return err
				}
				_, err := fmt.Fprintf(w, "%d/%d process · tổng %s\n", len(list), n, fsutil.FormatBytes(total))
				return err
			})
		},
	}
	c.Flags().String("sort", "mem", "mem|name|pid")
	c.Flags().Int("top", 30, "Số dòng (0 = tất cả)")
	c.Flags().Bool("group", false, "Gộp theo tên")
	rootCmd.AddCommand(c)
}
