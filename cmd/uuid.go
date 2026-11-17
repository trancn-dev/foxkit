package cmd

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"uuid"

	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "uuid",
		Short: "Sinh UUID v4 hoặc v7",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			v, _ := c.Flags().GetString("v")
			count, _ := c.Flags().GetInt("count")
			upper, _ := c.Flags().GetBool("upper")
			noDash, _ := c.Flags().GetBool("no-dash")
			if v != "4" && v != "7" {
				return output.Usage("UUID_VERSION_INVALID", "Phiên bản UUID phải là 4 hoặc 7", "Dùng --v 4 hoặc --v 7")
			}
			if count < 1 || count > 1000 {
				return output.Usage("UUID_COUNT_INVALID", "Số lượng phải từ 1 đến 1000", "Ví dụ: --count 10")
			}
			items := make([]string, count)
			for i := range items {
				if v == "7" {
					items[i] = uuid.NewV7().String()
				} else {
					items[i] = uuid.NewV4().String()
				}
			}
			sort.Strings(items) // v7 sinh trong cùng 1 ms vẫn tăng dần; v4 thì thứ tự vô nghĩa
			for i, s := range items {
				if noDash {
					s = strings.ReplaceAll(s, "-", "")
				}
				if upper {
					s = strings.ToUpper(s)
				}
				items[i] = s
			}
			ver := 4
			if v == "7" {
				ver = 7
			}
			return printer(c).Success(map[string]any{"version": ver, "count": count, "items": items}, func(w io.Writer) error {
				_, err := fmt.Fprintln(w, strings.Join(items, "\n"))
				return err
			})
		},
	}
	c.Flags().String("v", "4", "Phiên bản UUID: 4 hoặc 7")
	c.Flags().Int("count", 1, "Số lượng (1-1000)")
	c.Flags().Bool("upper", false, "Chữ hoa")
	c.Flags().Bool("no-dash", false, "Bỏ dấu gạch")
	rootCmd.AddCommand(c)
}
