package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"foxkit/internal/env"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	parent := &cobra.Command{Use: "env", Short: "Kiểm tra file .env"}

	diff := &cobra.Command{
		Use:   "diff [dir]",
		Short: "So .env với .env.example: thiếu, thừa, rỗng",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			file, _ := c.Flags().GetString("file")
			example, _ := c.Flags().GetString("example")
			dir, err := envDir(args, 0)
			if err != nil {
				return err
			}
			envf, err := loadEnv(dir, file, "ENV_FILE_NOT_FOUND", "Tạo bằng: copy "+example+" "+file)
			if err != nil {
				return err
			}
			exf, err := loadEnv(dir, example, "ENV_EXAMPLE_NOT_FOUND", "Dùng --example để chỉ file mẫu khác")
			if err != nil {
				return err
			}
			d := env.Compare(envf, exf)
			data := map[string]any{
				"file": file, "example": example,
				"missing": d.Missing, "extra": d.Extra, "empty": d.Empty, "duplicates": envf.Duplicates(),
			}
			return printer(c).Success(data, func(w io.Writer) error {
				if len(d.Missing)+len(d.Extra)+len(d.Empty) == 0 {
					_, err := fmt.Fprintf(w, "%s khớp với %s ✓\n", file, example)
					return err
				}
				fmt.Fprintf(w, "%s so với %s\n", file, example)
				for _, row := range []struct {
					label string
					keys  []string
				}{{"Thiếu", d.Missing}, {"Thừa ", d.Extra}, {"Rỗng ", d.Empty}} {
					if len(row.keys) > 0 {
						fmt.Fprintf(w, "%s (%d):  %s\n", row.label, len(row.keys), strings.Join(row.keys, ", "))
					}
				}
				if dups := envf.Duplicates(); len(dups) > 0 {
					fmt.Fprintf(w, "Cảnh báo: key khai báo trùng: %s\n", strings.Join(dups, ", "))
				}
				return nil
			})
		},
	}
	diff.Flags().String("file", ".env", "File env")
	diff.Flags().String("example", ".env.example", "File mẫu")

	get := &cobra.Command{
		Use:   "get <KEY> [dir]",
		Short: "Đọc giá trị một key trong .env",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(c *cobra.Command, args []string) error {
			file, _ := c.Flags().GetString("file")
			show, _ := c.Flags().GetBool("show")
			dir, err := envDir(args, 1)
			if err != nil {
				return err
			}
			envf, err := loadEnv(dir, file, "ENV_FILE_NOT_FOUND", "Tạo bằng: copy .env.example .env")
			if err != nil {
				return err
			}
			key := args[0]
			val, ok := envf.Get(key)
			if !ok {
				return output.Fail("ENV_KEY_NOT_FOUND", "Không thấy key "+key+" trong "+file, "Xem danh sách key bằng foxkit env diff")
			}
			masked := env.IsSecretKey(key) && !show
			if masked {
				val = env.Mask(val)
			}
			data := map[string]any{"key": key, "value": val, "masked": masked, "line": envf.LineOf(key)}
			return printer(c).Success(data, func(w io.Writer) error {
				fmt.Fprintln(w, val)
				if strings.Contains(val, "${") {
					fmt.Fprintln(c.ErrOrStderr(), "Lưu ý: foxkit không expand ${VAR}, giá trị hiển thị là nguyên văn.")
				}
				return nil
			})
		},
	}
	get.Flags().String("file", ".env", "File env")
	get.Flags().Bool("show", false, "Hiện giá trị bí mật")

	parent.AddCommand(diff, get)
	rootCmd.AddCommand(parent)
}

// envDir lấy thư mục dự án từ đối số thứ idx (mặc định thư mục hiện tại) và kiểm tra tồn tại.
func envDir(args []string, idx int) (string, error) {
	dir := "."
	if len(args) > idx {
		dir = args[idx]
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", output.DirNotFound(dir)
	}
	return dir, nil
}

func loadEnv(dir, name, notFoundCode, hint string) (*env.File, error) {
	p := name
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, name)
	}
	f, err := env.ParseFile(p)
	var pe *env.ParseError
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, output.Fail(notFoundCode, "Không tìm thấy file "+p, hint)
	case errors.As(err, &pe):
		return nil, output.Fail("ENV_PARSE_FAILED", fmt.Sprintf("Sai cú pháp trong %s, %s", p, pe.Error()), fmt.Sprintf("Kiểm tra dòng %d trong %s", pe.Line, p))
	case err != nil:
		return nil, output.Internal(err)
	}
	return f, nil
}
