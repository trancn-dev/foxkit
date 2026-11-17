package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"foxkit/internal/fsutil"
	"foxkit/internal/jsonx"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

const jsonLimit = 50 << 20

func init() {
	c := &cobra.Command{
		Use:   "json [file|-]",
		Short: "Format, minify, kiểm tra, lấy giá trị JSON",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			text, _ := f.GetString("text")
			min, _ := f.GetBool("min")
			indentStr, _ := f.GetString("indent")
			sortKeys, _ := f.GetBool("sort-keys")
			get, _ := f.GetString("get")
			validate, _ := f.GetBool("validate")

			var data []byte
			var err error
			switch {
			case f.Changed("text"):
				data = []byte(text)
			case len(args) == 1 && args[0] == "-":
				if data, err = readStdin(c, jsonLimit, "50MB"); err != nil {
					return err
				}
			case len(args) == 1:
				st, serr := os.Stat(args[0])
				if serr != nil {
					return output.FileNotFound(args[0])
				}
				if st.Size() > jsonLimit {
					return output.InputTooLarge("50MB")
				}
				if data, err = os.ReadFile(args[0]); err != nil {
					return output.Internal(err)
				}
			default:
				return output.Usage("JSON_NO_INPUT", "Thiếu file hoặc --text", "Ví dụ: foxkit json data.json")
			}
			if min && (f.Changed("indent") || sortKeys || get != "") {
				return output.Usage("JSON_ARGS_CONFLICT", "--min không dùng chung với --indent, --sort-keys hoặc --get", "")
			}
			indent := "  "
			switch {
			case indentStr == "tab":
				indent = "\t"
			default:
				n, perr := strconv.Atoi(indentStr)
				if perr != nil || n < 0 || n > 8 {
					return output.Usage("ARGS_INVALID", "--indent phải từ 0 đến 8 hoặc tab", "")
				}
				indent = string(make([]byte, 0))
				for i := 0; i < n; i++ {
					indent += " "
				}
			}

			if se := jsonx.Validate(data); se != nil {
				e := output.Fail("JSON_INVALID", se.Error()+"\n"+se.Context, fmt.Sprintf("Sửa tại dòng %d cột %d", se.Line, se.Col))
				e.Position = &output.Position{Line: se.Line, Col: se.Col}
				return e
			}
			p := printer(c)
			if validate {
				return p.Success(map[string]any{"valid": true}, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "JSON hợp lệ (%s).\n", fsutil.FormatBytes(int64(len(data))))
					return err
				})
			}
			if get != "" {
				v, err := jsonx.Get(data, get)
				var pe *jsonx.PathError
				var nf *jsonx.NotFound
				switch {
				case errors.As(err, &pe):
					return output.Usage("JSON_PATH_INVALID", "Đường dẫn sai: "+pe.Msg, "Ví dụ: data.items.0.id")
				case errors.As(err, &nf):
					return output.Fail("JSON_PATH_NOT_FOUND", nf.Error(), "Kiểm tra lại tên key hoặc chỉ số")
				case err != nil:
					return output.Internal(err)
				}
				return p.Success(map[string]any{"valid": true, "result": v}, func(w io.Writer) error {
					if s, ok := v.(string); ok {
						_, err := fmt.Fprintln(w, s) // chuỗi in trần để dùng trong script
						return err
					}
					b, err := jsonx.Encode(v, "  ")
					if err != nil {
						return err
					}
					_, err = fmt.Fprintln(w, string(b))
					return err
				})
			}
			var out []byte
			switch {
			case min:
				out, err = jsonx.Minify(data)
			case sortKeys:
				out, err = jsonx.SortKeys(data, indent)
			default:
				out, err = jsonx.Pretty(data, indent)
			}
			if err != nil {
				return output.Internal(err)
			}
			return p.Success(map[string]any{"valid": true, "result": string(out)}, func(w io.Writer) error {
				_, err := fmt.Fprintln(w, string(out))
				return err
			})
		},
	}
	c.Flags().String("text", "", "JSON truyền thẳng")
	c.Flags().Bool("min", false, "Minify")
	c.Flags().String("indent", "2", "Số khoảng trắng 0-8 hoặc tab")
	c.Flags().Bool("sort-keys", false, "Sắp key theo ABC")
	c.Flags().String("get", "", "Lấy giá trị theo đường dẫn")
	c.Flags().Bool("validate", false, "Chỉ kiểm tra")
	rootCmd.AddCommand(c)
}
