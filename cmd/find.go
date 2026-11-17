package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"foxkit/internal/find"
	"foxkit/internal/fsutil"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "find [pattern] [dir]",
		Short: "Tìm file theo tên, đuôi, kích thước, ngày, nội dung",
		Args:  cobra.MaximumNArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			var pattern, dir string
			dir = "."
			switch len(args) {
			case 1:
				// một đối số trông như đường dẫn thư mục có thật (có dấu \ / hoặc ổ đĩa) thì hiểu là dir
				if strings.ContainsAny(args[0], `\/:`) && fsutil.IsDir(args[0]) {
					dir = args[0]
				} else {
					pattern = args[0]
				}
			case 2:
				pattern, dir = args[0], args[1]
			}
			opt := find.Options{Dir: dir, Pattern: pattern, Now: time.Now()}
			opt.Exts, _ = f.GetStringSlice("ext")
			opt.Type, _ = f.GetString("type")
			opt.Content, _ = f.GetString("content")
			opt.IgnoreCase, _ = f.GetBool("ignore-case")
			opt.Max, _ = f.GetInt("max")
			opt.Hidden, _ = f.GetBool("hidden")
			parseSize := func(name string) (*int64, error) {
				s, _ := f.GetString(name)
				if s == "" {
					return nil, nil
				}
				n, err := fsutil.ParseSize(s)
				if err != nil {
					return nil, output.SizeInvalid(s)
				}
				return &n, nil
			}
			parseAge := func(name string) (time.Duration, error) {
				s, _ := f.GetString(name)
				if s == "" {
					return 0, nil
				}
				d, err := fsutil.ParseAge(s)
				if err != nil {
					return 0, output.AgeInvalid(s)
				}
				return d, nil
			}
			var err error
			if opt.Larger, err = parseSize("larger"); err != nil {
				return err
			}
			if opt.Smaller, err = parseSize("smaller"); err != nil {
				return err
			}
			if opt.Newer, err = parseAge("newer"); err != nil {
				return err
			}
			if opt.Older, err = parseAge("older"); err != nil {
				return err
			}
			if opt.Type != "f" && opt.Type != "d" && opt.Type != "a" {
				return output.Usage("ARGS_INVALID", "--type phải là f, d hoặc a", "")
			}
			if opt.Max < 1 || opt.Max > 10000 {
				return output.Usage("ARGS_INVALID", "--max phải từ 1 đến 10000", "")
			}
			if pattern != "" && !find.ValidPattern(pattern) {
				return output.Usage("PATTERN_INVALID", "Mẫu tên không hợp lệ: "+pattern, `Ví dụ: "*.pdf" hoặc "hop dong"`)
			}
			if pattern == "" && len(opt.Exts) == 0 && opt.Larger == nil && opt.Smaller == nil && opt.Newer == 0 && opt.Older == 0 && opt.Content == "" && !f.Changed("type") {
				return output.Usage("FIND_NO_CRITERIA", "Chưa có điều kiện tìm nào", "Cần ít nhất một điều kiện tìm")
			}
			if !fsutil.IsDir(dir) {
				return output.DirNotFound(dir)
			}
			ctx, stop := cmdContext(c)
			defer stop()
			res, err := find.Run(ctx, opt)
			if err != nil {
				return canceled(err)
			}
			return printer(c).Success(res, func(w io.Writer) error {
				for _, h := range res.Hits {
					switch {
					case h.Line > 0:
						fmt.Fprintf(w, "%s:%d   %s\n", h.Path, h.Line, h.Snippet)
					case h.IsDir:
						fmt.Fprintf(w, "%s%c  %s\n", h.Path, os.PathSeparator, h.Modified.Format("2006-01-02"))
					default:
						fmt.Fprintf(w, "%-60s %9s   %s\n", h.Path, fsutil.FormatBytes(h.Bytes), h.Modified.Format("2006-01-02"))
					}
				}
				more := ""
				if res.Truncated {
					more = " (đã chạm --max, còn nữa)"
				}
				_, err := fmt.Fprintf(w, "%d kết quả (đã quét %s mục)%s\n", len(res.Hits), fsutil.FormatInt(res.Scanned), more)
				return err
			})
		},
	}
	f := c.Flags()
	f.StringSlice("ext", nil, "Đuôi file")
	f.String("type", "f", "f|d|a")
	f.String("larger", "", "Lớn hơn")
	f.String("smaller", "", "Nhỏ hơn")
	f.String("newer", "", "Sửa trong vòng")
	f.String("older", "", "Cũ hơn")
	f.String("content", "", "Chứa nội dung")
	f.Bool("ignore-case", true, "Không phân biệt hoa thường (nội dung)")
	f.Int("max", 200, "Tối đa kết quả")
	f.Bool("hidden", false, "Gồm file ẩn")
	rootCmd.AddCommand(c)
}
