package cmd

import (
	"fmt"
	"io"
	"slices"
	"sort"

	"foxkit/internal/output"
	"foxkit/internal/textcase"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "case <mode> <text|->",
		Short: "camel, snake, kebab, studly, slug tiếng Việt...",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			mode := args[0]
			if mode != "all" && mode != "pascal" && !slices.Contains(textcase.Modes, mode) {
				return output.Usage("CASE_MODE_INVALID", "Kiểu không hợp lệ: "+mode, "Chọn: camel, snake, kebab, studly, title, upper, lower, slug, constant, all")
			}
			if len(args) < 2 {
				return output.Usage("CASE_NO_INPUT", "Thiếu chuỗi cần đổi", `Ví dụ: foxkit case snake "userName"`)
			}
			in := args[1]
			if in == "-" {
				b, err := io.ReadAll(io.LimitReader(c.InOrStdin(), 1<<20))
				if err != nil {
					return output.Internal(err)
				}
				in = string(b)
			}
			if mode == "all" {
				res := textcase.All(in)
				return printer(c).Success(map[string]any{"input": in, "mode": mode, "result": res}, func(w io.Writer) error {
					keys := make([]string, 0, len(res))
					for k := range res {
						keys = append(keys, k)
					}
					sort.Strings(keys)
					for _, k := range keys {
						fmt.Fprintf(w, "%-9s %s\n", k, res[k])
					}
					return nil
				})
			}
			res, err := textcase.Convert(mode, in)
			if err != nil {
				return output.Internal(err)
			}
			return printer(c).Success(map[string]any{"input": in, "mode": mode, "result": res}, func(w io.Writer) error {
				_, err := fmt.Fprintln(w, res)
				return err
			})
		},
	}
	rootCmd.AddCommand(c)
}
