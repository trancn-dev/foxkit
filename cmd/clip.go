package cmd

import (
	"fmt"
	"io"
	"unicode/utf8"

	"foxkit/internal/clip"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "clip [text|-]",
		Short: "Đọc/ghi clipboard",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			clear, _ := c.Flags().GetBool("clear")
			p := printer(c)
			write := func(text string, fromStdin bool) error {
				if fromStdin {
					text = clip.TrimOneNewline(text)
				}
				text = clip.NormalizeCRLF(text)
				if len(text) > clip.MaxBytes {
					return output.InputTooLarge("10MB")
				}
				if err := clip.Retry(func() error { return clip.Default.Write(text) }); err != nil {
					return output.Fail("CLIP_BUSY", "Clipboard đang bị ứng dụng khác giữ", "Đóng ứng dụng đang giữ clipboard rồi thử lại")
				}
				return p.Success(map[string]any{"written": true, "length": utf8.RuneCountInString(text)}, func(w io.Writer) error {
					_, err := fmt.Fprintf(w, "Đã ghi %d ký tự vào clipboard.\n", utf8.RuneCountInString(text))
					return err
				})
			}
			switch {
			case clear:
				return write("", false)
			case len(args) == 1 && args[0] == "-":
				b, err := readStdin(c, clip.MaxBytes, "10MB")
				if err != nil {
					return err
				}
				return write(string(b), true)
			case len(args) == 1:
				return write(args[0], false)
			}
			var text string
			err := clip.Retry(func() (err error) { text, err = clip.Default.Read(); return })
			if err != nil {
				return output.Fail("CLIP_NO_TEXT", "Không đọc được chữ từ clipboard", "Clipboard đang chứa ảnh hoặc file, không phải chữ (hoặc đang bị ứng dụng khác giữ)")
			}
			return p.Success(map[string]any{"text": text, "length": utf8.RuneCountInString(text)}, func(w io.Writer) error {
				// pipe tiếp thì in nguyên văn; là terminal thì thêm xuống dòng cuối nếu chưa có
				if isTerminal(c.OutOrStdout()) && len(text) > 0 && text[len(text)-1] != '\n' {
					text += "\n"
				}
				_, err := io.WriteString(w, text)
				return err
			})
		},
	}
	c.Flags().Bool("clear", false, "Xóa clipboard")
	rootCmd.AddCommand(c)
}
