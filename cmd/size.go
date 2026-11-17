package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/size"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "size [dir]",
		Short: "Thư mục/file nào chiếm nhiều dung lượng nhất",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			top, _ := c.Flags().GetInt("top")
			files, _ := c.Flags().GetBool("files")
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if top < 1 || top > 500 {
				return output.Usage("TOP_INVALID", "--top phải từ 1 đến 500", "Ví dụ: --top 50")
			}
			if !fsutil.IsDir(dir) {
				return output.DirNotFound(dir)
			}
			abs, err := filepath.Abs(dir)
			if err != nil {
				return output.Internal(err)
			}
			ctx, stop := cmdContext(c)
			defer stop()
			var rep size.Report
			if files {
				rep, err = size.LargestFiles(ctx, abs, top)
			} else {
				rep, err = size.ByChildren(ctx, abs, top)
			}
			if err != nil {
				return canceled(err)
			}
			return printer(c).Success(rep, func(w io.Writer) error {
				fmt.Fprintf(w, "%s  — %s\n\n", rep.Path, fsutil.FormatBytes(rep.Total))
				for _, it := range rep.Items {
					filled := int(it.Pct/100*20 + 0.5)
					bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
					name := it.Name
					if files {
						name = it.Path
					} else if it.IsDir {
						name += string(os.PathSeparator)
					}
					fmt.Fprintf(w, "  %9s  %5.1f%%  %s  %s\n", fsutil.FormatBytes(it.Bytes), it.Pct, bar, name)
				}
				if rep.Denied > 0 {
					fmt.Fprintf(w, "(%d mục không đọc được do thiếu quyền)\n", rep.Denied)
				}
				return nil
			})
		},
	}
	c.Flags().Int("top", 20, "Số mục (1-500)")
	c.Flags().Bool("files", false, "Liệt kê file lớn nhất trong cả cây")
	rootCmd.AddCommand(c)
}
