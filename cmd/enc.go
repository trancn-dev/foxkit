package cmd

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"slices"
	"unicode/utf8"

	"foxkit/internal/enc"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

const encLimit = 10 << 20

func init() {
	c := &cobra.Command{
		Use:   "enc <mode> [text|-]",
		Short: "base64, base64url, hex, url, html",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			mode := args[0]
			decode, _ := c.Flags().GetBool("decode")
			file, _ := c.Flags().GetString("file")
			if !slices.Contains(enc.Modes, mode) {
				return output.Usage("ENC_MODE_INVALID", "Kiểu không hợp lệ: "+mode, "Chọn: base64, base64url, hex, url, html")
			}
			var in []byte
			switch {
			case file != "":
				b, err := os.ReadFile(file)
				if err != nil {
					return output.FileNotFound(file)
				}
				if len(b) > encLimit {
					return encTooLarge()
				}
				in = b
			case len(args) == 2 && args[1] == "-":
				b, err := io.ReadAll(io.LimitReader(c.InOrStdin(), encLimit+1))
				if err != nil {
					return output.Internal(err)
				}
				if len(b) > encLimit {
					return encTooLarge()
				}
				in = b
			case len(args) == 2:
				in = []byte(args[1])
			default:
				return output.Usage("ENC_NO_INPUT", "Thiếu nội dung cần xử lý", `Ví dụ: foxkit enc base64 "hello"`)
			}

			var result string
			binary := false
			if decode {
				out, err := enc.Decode(mode, string(in))
				if err != nil {
					return output.Fail("ENC_DECODE_FAILED", "Không giải mã được: "+err.Error(), "Kiểm tra chuỗi có đúng định dạng "+mode+" không")
				}
				if utf8.Valid(out) {
					result = string(out)
				} else {
					binary = true
					result = base64.StdEncoding.EncodeToString(out) // JSON: result là base64 của dữ liệu nhị phân
					if !printer(c).JSON {
						fmt.Fprintln(c.ErrOrStderr(), "Kết quả là dữ liệu nhị phân")
						result = fmt.Sprintf("%x", out)
					}
				}
			} else {
				var err error
				if result, err = enc.Encode(mode, in); err != nil {
					return output.Internal(err)
				}
			}
			data := map[string]any{"mode": mode, "decode": decode, "binary": binary, "result": result}
			return printer(c).Success(data, func(w io.Writer) error {
				if isTerminal(c.OutOrStdout()) {
					result += "\n"
				}
				_, err := io.WriteString(w, result)
				return err
			})
		},
	}
	c.Flags().BoolP("decode", "d", false, "Giải mã thay vì mã hóa")
	c.Flags().String("file", "", "Đọc nội dung từ file")
	rootCmd.AddCommand(c)
}

func encTooLarge() error {
	e := output.InputTooLarge("10MB")
	e.Code, e.Hint = "ENC_INPUT_TOO_LARGE", "Dùng --file với file nhỏ hơn 10MB"
	return e
}

// isTerminal: w có phải console thật (không phải pipe/file/buffer test) không.
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && st.Mode()&os.ModeCharDevice != 0
}
