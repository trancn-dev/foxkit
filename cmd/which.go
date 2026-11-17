package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"foxkit/internal/output"
	"foxkit/internal/which"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "which <name>...",
		Short: "Liệt kê mọi bản exe trong PATH, bản nào đang được dùng",
		RunE: func(c *cobra.Command, names []string) error {
			withVersion, _ := c.Flags().GetBool("version")
			if len(names) == 0 {
				return output.Usage("WHICH_NO_NAME", "Thiếu tên lệnh cần tìm", "Ví dụ: foxkit which php")
			}
			cwd, _ := os.Getwd() // không lấy được cwd thì chỉ tìm trong PATH
			items := make([]which.Result, 0, len(names))
			var missing []string
			for _, n := range names {
				ms := which.Find(n, os.Getenv("PATH"), os.Getenv("PATHEXT"), cwd)
				if len(ms) == 0 {
					missing = append(missing, n)
				} else if withVersion {
					which.Versions(c.Context(), ms)
				}
				items = append(items, which.Result{Name: n, Matches: ms})
			}
			perr := printer(c).Success(map[string]any{"items": items}, func(w io.Writer) error {
				for _, it := range items {
					fmt.Fprintln(w, it.Name)
					for _, m := range it.Matches {
						mark := "  "
						if m.Active {
							mark = "✓ "
						}
						note := ""
						if strings.EqualFold(filepath.Clean(m.Dir), filepath.Clean(cwd)) {
							note = "  (thư mục hiện tại: cmd.exe tìm ở đây trước, PowerShell thì không)"
						}
						fmt.Fprintf(w, "  %s%-50s %s%s\n", mark, m.Path, m.Version, note)
					}
					if len(it.Matches) == 0 {
						fmt.Fprintln(w, "  (không có trong PATH)")
					} else if len(it.Matches) > 1 {
						fmt.Fprintln(w, "Bản đang dùng là bản đầu tiên. Muốn đổi thì sắp xếp lại PATH (foxkit path).")
					}
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			if len(missing) > 0 {
				n := missing[0]
				return output.Fail("WHICH_NOT_FOUND", strings.Join(missing, ", ")+" không có trong PATH", n+" chưa có trong PATH. Kiểm tra cài đặt hoặc thêm thư mục chứa nó vào PATH")
			}
			return nil
		},
	}
	c.Flags().Bool("version", false, "Hiện phiên bản từng bản")
	rootCmd.AddCommand(c)
}
