package cmd

import (
	"fmt"
	"io"
	"strings"

	"foxkit/internal/clip"
	"foxkit/internal/output"
	"foxkit/internal/pass"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "createpw",
		Short: "Sinh mật khẩu 16 ký tự (A-Z a-z 0-9 ký hiệu), bắt đầu bằng chữ hoa",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			n, _ := c.Flags().GetInt("len")
			count, _ := c.Flags().GetInt("count")
			noAmb, _ := c.Flags().GetBool("no-ambiguous")
			doCopy, _ := c.Flags().GetBool("copy")
			if n < 4 || n > 256 {
				return output.Usage("PASS_LEN_INVALID", "Độ dài phải từ 4 đến 256", "Ví dụ: --len 20")
			}
			if count < 1 || count > 100 {
				return output.Usage("PASS_COUNT_INVALID", "Số lượng phải từ 1 đến 100", "Ví dụ: --count 5")
			}
			items := make([]string, count)
			for i := range items {
				var err error
				if items[i], err = pass.GenerateStartUpper(n, noAmb); err != nil {
					return output.Fail("PASS_RANDOM_FAILED", err.Error(), "Thử chạy lại")
				}
			}
			if doCopy {
				if err := clip.Retry(func() error { return clip.Default.Write(items[0]) }); err != nil {
					return output.Fail("CLIP_WRITE_FAILED", "Không ghi được clipboard: "+err.Error(), "Chạy lại không có --copy rồi tự copy")
				}
			}
			return printer(c).Success(map[string]any{"length": n, "copied": doCopy, "items": items}, func(w io.Writer) error {
				if _, err := fmt.Fprintln(w, strings.Join(items, "\n")); err != nil {
					return err
				}
				if doCopy {
					fmt.Fprintln(c.ErrOrStderr(), "Đã copy mật khẩu đầu tiên vào clipboard")
				}
				return nil
			})
		},
	}
	c.Flags().Int("len", 16, "Độ dài (4-256)")
	c.Flags().Int("count", 1, "Số lượng (1-100)")
	c.Flags().Bool("no-ambiguous", false, "Bỏ ký tự dễ nhầm (0 O o 1 l I |)")
	c.Flags().Bool("copy", false, "Copy mật khẩu đầu tiên vào clipboard")
	rootCmd.AddCommand(c)
}
