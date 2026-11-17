package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"slices"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/rename"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "rename <dir>",
		Short: "Tìm-thay thế, đổi kiểu chữ, đánh số tên file",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			opt := rename.Options{Dir: args[0]}
			opt.Find, _ = f.GetString("find")
			opt.Replace, _ = f.GetString("replace")
			opt.Regex, _ = f.GetBool("regex")
			opt.IgnoreCase, _ = f.GetBool("ignore-case")
			opt.WithExt, _ = f.GetBool("with-ext")
			opt.Case, _ = f.GetString("case")
			opt.Number, _ = f.GetString("number")
			opt.Sort, _ = f.GetString("sort")
			opt.Start, _ = f.GetInt("start")
			opt.Exts, _ = f.GetStringSlice("ext")
			opt.Recursive, _ = f.GetBool("recursive")
			opt.Dirs, _ = f.GetBool("dirs")
			apply, _ := f.GetBool("apply")

			modes := 0
			for _, on := range []bool{opt.Find != "", opt.Case != "", opt.Number != ""} {
				if on {
					modes++
				}
			}
			switch {
			case modes == 0:
				return output.Usage("RENAME_MODE_REQUIRED", "Chưa chọn chế độ đổi tên", "Dùng --find/--replace, --case hoặc --number")
			case modes > 1:
				return output.Usage("RENAME_MODE_CONFLICT", "Chỉ chọn một chế độ: --find/--replace, --case hoặc --number", "")
			}
			if !fsutil.IsDir(opt.Dir) {
				return output.DirNotFound(opt.Dir)
			}
			if opt.Case != "" && !slices.Contains([]string{"lower", "upper", "slug", "snake", "kebab"}, opt.Case) {
				return output.Usage("ARGS_INVALID", "--case phải là lower, upper, slug, snake hoặc kebab", "")
			}
			if opt.Sort != "" && !slices.Contains([]string{"name", "date", "size"}, opt.Sort) {
				return output.Usage("ARGS_INVALID", "--sort phải là name, date hoặc size", "")
			}
			if opt.Number != "" && !rename.ValidatePattern(opt.Number) {
				return output.Usage("RENAME_PATTERN_INVALID", "Mẫu --number thiếu {n}", `Ví dụ: --number "Trip_{n:3}"`)
			}

			plan, err := rename.Build(opt)
			if err != nil {
				if opt.Find != "" && isRegexErr(err) {
					return output.Usage("REGEX_INVALID", "Biểu thức regex không hợp lệ: "+err.Error(), "Go dùng RE2: không hỗ trợ lookbehind (?<=)")
				}
				return output.Internal(err)
			}
			if apply {
				if len(plan.Conflicts) > 0 {
					return output.Fail("RENAME_CONFLICT", fmt.Sprintf("Có %d xung đột, không đổi tên gì cả", len(plan.Conflicts)), "Xem mục conflicts, sửa mẫu rồi chạy lại")
				}
				if _, err := rename.Apply(plan); err != nil {
					return output.Fail("RENAME_FAILED", "Đổi tên lỗi giữa chừng (đã hoàn tác): "+err.Error(), "File có thể đang mở trong chương trình khác, thử foxkit lock")
				}
				plan.Applied = true
			}
			return printer(c).Success(plan, func(w io.Writer) error {
				for _, ch := range plan.Changes {
					rel, rerr := filepath.Rel(opt.Dir, filepath.Join(ch.Dir, ch.From))
					if rerr != nil {
						rel = ch.From
					}
					fmt.Fprintf(w, "%s  →  %s\n", rel, ch.To)
				}
				for _, cf := range plan.Conflicts {
					fmt.Fprintf(w, "XUNG ĐỘT (%s): %s\n", cf.Reason, cf.To)
				}
				if plan.Applied {
					fmt.Fprintf(w, "Đã đổi tên %d file, %d file giữ nguyên.\n", len(plan.Changes), plan.Unchanged)
					return nil
				}
				fmt.Fprintf(w, "%d file sẽ đổi tên, %d file giữ nguyên.\n%s\n", len(plan.Changes), plan.Unchanged, output.DryRunNote)
				return nil
			})
		},
	}
	f := c.Flags()
	f.String("find", "", "Chuỗi/regex cần tìm")
	f.String("replace", "", "Thay bằng")
	f.Bool("regex", false, "Dùng regex (RE2)")
	f.Bool("ignore-case", false, "Không phân biệt hoa thường")
	f.Bool("with-ext", false, "Áp dụng cả đuôi file")
	f.String("case", "", "lower|upper|slug|snake|kebab")
	f.String("number", "", "Mẫu đánh số, ví dụ Trip_{n:3}")
	f.String("sort", "name", "name|date|size")
	f.Int("start", 1, "Số bắt đầu")
	f.StringSlice("ext", nil, "Chỉ đuôi file")
	f.Bool("recursive", false, "Gồm thư mục con")
	f.Bool("dirs", false, "Đổi tên cả thư mục")
	f.Bool("apply", false, "Thực hiện")
	rootCmd.AddCommand(c)
}

// isRegexErr: lỗi biên dịch regexp (chuỗi lỗi của RE2 luôn bắt đầu bằng "error parsing regexp").
func isRegexErr(err error) bool {
	return regexp.MustCompile(`^error parsing regexp`).MatchString(err.Error())
}
