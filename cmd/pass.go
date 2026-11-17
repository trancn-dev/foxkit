package cmd

import (
	"fmt"
	"io"
	"math"
	"strings"

	"foxkit/internal/output"
	"foxkit/internal/pass"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "pass",
		Short: "Sinh mật khẩu/chuỗi ngẫu nhiên an toàn",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			n, _ := c.Flags().GetInt("len")
			count, _ := c.Flags().GetInt("count")
			charset, _ := c.Flags().GetString("charset")
			noAmb, _ := c.Flags().GetBool("no-ambiguous")
			if n < 4 || n > 1024 {
				return output.Usage("PASS_LEN_INVALID", "Độ dài phải từ 4 đến 1024", "Ví dụ: --len 32")
			}
			if count < 1 || count > 100 {
				return output.Usage("PASS_COUNT_INVALID", "Số lượng phải từ 1 đến 100", "Ví dụ: --count 5")
			}
			alpha, err := pass.Alphabet(charset, noAmb)
			if err != nil {
				return output.Usage("PASS_CHARSET_INVALID", "Bộ ký tự không hợp lệ: "+charset, "Chọn: all, alnum, alpha, num, hex")
			}
			items := make([]string, count)
			for i := range items {
				if items[i], err = pass.Generate(pass.Options{Length: n, Charset: charset, NoAmbiguous: noAmb}); err != nil {
					return output.Fail("PASS_RANDOM_FAILED", err.Error(), "Thử chạy lại")
				}
			}
			bits := math.Round(pass.EntropyBits(n, len(alpha))*10) / 10
			data := map[string]any{"length": n, "charset": charset, "entropyBits": bits, "items": items}
			return printer(c).Success(data, func(w io.Writer) error {
				_, err := fmt.Fprintln(w, strings.Join(items, "\n"))
				return err
			})
		},
	}
	c.Flags().Int("len", 24, "Độ dài (4-1024)")
	c.Flags().Int("count", 1, "Số lượng (1-100)")
	c.Flags().String("charset", "all", "all|alnum|alpha|num|hex")
	c.Flags().Bool("no-ambiguous", false, "Bỏ ký tự dễ nhầm")
	rootCmd.AddCommand(c)
}
