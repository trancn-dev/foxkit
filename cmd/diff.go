package cmd

import (
	"fmt"
	"io"
	"os"

	"foxkit/internal/diff"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "diff <a> <b>",
		Short: "So sánh hai file hoặc hai thư mục",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			opt := diff.Options{}
			opt.Context, _ = f.GetInt("context")
			opt.IgnoreSpace, _ = f.GetBool("ignore-space")
			opt.IgnoreEOL, _ = f.GetBool("ignore-eol")
			opt.IgnoreCase, _ = f.GetBool("ignore-case")
			opt.NamesOnly, _ = f.GetBool("names-only")
			opt.Exclude, _ = f.GetStringSlice("exclude")
			exitCode, _ := f.GetBool("exit-code")
			if opt.Context < 0 || opt.Context > 100 {
				return output.Usage("ARGS_INVALID", "--context phải từ 0 đến 100", "")
			}
			sa, err := os.Stat(args[0])
			if err != nil {
				return output.Usage("DIFF_NOT_FOUND", "Không tìm thấy "+args[0], "Kiểm tra lại đường dẫn")
			}
			sb, err := os.Stat(args[1])
			if err != nil {
				return output.Usage("DIFF_NOT_FOUND", "Không tìm thấy "+args[1], "Kiểm tra lại đường dẫn")
			}
			if sa.IsDir() != sb.IsDir() {
				return output.Usage("DIFF_TYPE_MISMATCH", "Hai đối số phải cùng là file hoặc cùng là thư mục", "")
			}
			var res diff.Result
			if sa.IsDir() {
				if res, err = diff.Dirs(args[0], args[1], opt); err != nil {
					return output.Internal(err)
				}
			} else {
				fd, err := diff.Files(args[0], args[1], opt)
				if err != nil {
					return output.Internal(err)
				}
				fd.Path = args[1]
				res = diff.Result{A: args[0], B: args[1], Identical: fd.Status == "same", Files: []diff.FileDiff{fd}, Added: fd.Added, Removed: fd.Removed}
				if fd.Status != "same" {
					res.Changed = 1
				}
			}
			perr := printer(c).Success(res, func(w io.Writer) error {
				color := isTerminal(w)
				if !sa.IsDir() {
					fd := res.Files[0]
					switch fd.Status {
					case "same":
						_, err := fmt.Fprintln(w, "Hai file giống nhau.")
						return err
					case "binary", "changed":
						if len(fd.Hunks) == 0 {
							_, err := fmt.Fprintf(w, "File khác nhau (%s, không in nội dung).\n", fd.Status)
							return err
						}
					}
					_, err := io.WriteString(w, diff.Unified(args[0], args[1], fd.Hunks, color))
					return err
				}
				var mod, add, del, bin int
				for _, fd := range res.Files {
					switch fd.Status {
					case "added":
						add++
						fmt.Fprintf(w, "A  %s\n", fd.Path)
					case "removed":
						del++
						fmt.Fprintf(w, "D  %s\n", fd.Path)
					case "binary":
						bin++
						fmt.Fprintf(w, "B  %-30s (nhị phân, khác)\n", fd.Path)
					default:
						mod++
						if opt.NamesOnly {
							fmt.Fprintf(w, "M  %s\n", fd.Path)
						} else {
							fmt.Fprintf(w, "M  %-30s (+%d −%d)\n", fd.Path, fd.Added, fd.Removed)
						}
					}
				}
				if res.Identical {
					_, err := fmt.Fprintln(w, "Hai thư mục giống nhau.")
					return err
				}
				_, err := fmt.Fprintf(w, "Tổng: %d sửa · %d thêm · %d xóa · %d nhị phân\n", mod, add, del, bin)
				return err
			})
			if perr != nil {
				return perr
			}
			if exitCode && !res.Identical {
				return output.Fail("DIFF_DIFFERENT", "Có khác biệt", "")
			}
			return nil
		},
	}
	f := c.Flags()
	f.Int("context", 3, "Số dòng ngữ cảnh")
	f.Bool("ignore-space", false, "Bỏ qua khác biệt khoảng trắng")
	f.Bool("ignore-eol", true, "Bỏ qua CRLF/LF")
	f.Bool("ignore-case", false, "Bỏ qua hoa/thường")
	f.Bool("names-only", false, "Thư mục: chỉ liệt kê tên file khác")
	f.StringSlice("exclude", nil, "Glob tên bỏ qua")
	f.Bool("exit-code", false, "Exit 1 nếu có khác biệt")
	rootCmd.AddCommand(c)
}
