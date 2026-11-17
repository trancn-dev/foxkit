package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"foxkit/internal/env"
	"foxkit/internal/fsutil"
	"foxkit/internal/lara"
	"foxkit/internal/laralog"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

// laraDir lấy thư mục dự án từ đối số và kiểm tra là dự án Laravel.
func laraDir(args []string) (string, error) {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	if !fsutil.IsDir(dir) {
		return "", output.DirNotFound(dir)
	}
	if !lara.IsProject(dir) {
		return "", output.Usage("LARA_NOT_PROJECT", "Đây không phải dự án Laravel: "+dir, "Chạy trong thư mục dự án Laravel hoặc truyền đường dẫn")
	}
	return dir, nil
}

func printEntry(w io.Writer, e laralog.Entry) {
	t := e.Time
	if i := strings.IndexAny(t, " T"); i >= 0 && len(t) >= i+9 {
		t = t[i+1 : i+9]
	}
	count := ""
	if e.Count > 1 {
		count = fmt.Sprintf("  ×%d", e.Count)
	}
	fmt.Fprintf(w, "[%s] %-8s %s%s\n", t, e.Level, e.Message, count)
	if e.Exception != "" {
		fmt.Fprintf(w, "           %s  %s\n", e.Exception, e.File)
	}
	for _, fr := range e.Trace {
		fmt.Fprintf(w, "           %s\n", fr)
	}
}

func init() {
	parent := &cobra.Command{Use: "lara", Short: "Tiện ích Laravel: log, key, clear"}

	log := &cobra.Command{
		Use:   "log [dir]",
		Short: "Đọc laravel.log: lọc level, rút gọn trace",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			level, _ := f.GetString("level")
			last, _ := f.GetInt("last")
			sinceStr, _ := f.GetString("since")
			grep, _ := f.GetString("grep")
			full, _ := f.GetBool("full")
			follow, _ := f.GetBool("follow")
			dir, err := laraDir(args)
			if err != nil {
				return err
			}
			if level != "" && !slices.Contains(laralog.Levels, strings.ToLower(level)) {
				return output.Usage("LEVEL_INVALID", "Level không hợp lệ: "+level, "Chọn: "+strings.Join(laralog.Levels, ", "))
			}
			if last < 1 || last > 1000 {
				return output.Usage("ARGS_INVALID", "--last phải từ 1 đến 1000", "")
			}
			filter := laralog.Filter{MinLevel: level, Grep: grep, Full: full, Dir: dir}
			if sinceStr != "" {
				d, err := fsutil.ParseAge(sinceStr)
				if err != nil {
					return output.AgeInvalid(sinceStr)
				}
				filter.Since = time.Now().Add(-d)
			}
			files := lara.LogFiles(dir)
			if len(files) == 0 {
				return output.Fail("LARA_LOG_NOT_FOUND", "Chưa có file log trong storage/logs", "Chưa có lỗi nào được ghi, hoặc LOG_CHANNEL không phải single/daily")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			entries, err := laralog.ReadLast(files, filter, last)
			if err != nil {
				return output.Internal(err)
			}
			p := printer(c)
			if follow {
				for _, e := range entries {
					if p.JSON {
						_ = p.Event(e) // pipe đóng thì Follow cũng sắp dừng
					} else {
						printEntry(c.OutOrStdout(), e)
					}
				}
				emit := func(e laralog.Entry) {
					if p.JSON {
						_ = p.Event(e)
					} else {
						printEntry(c.OutOrStdout(), e)
					}
				}
				return canceled(laralog.Follow(ctx, func() string {
					if fs := lara.LogFiles(dir); len(fs) > 0 {
						return fs[0]
					}
					return files[0]
				}, filter, emit))
			}
			rel := make([]string, len(files))
			for i, fl := range files {
				rel[i] = strings.TrimPrefix(strings.ReplaceAll(fl, `\`, "/"), strings.ReplaceAll(dir, `\`, "/")+"/")
			}
			return p.Success(map[string]any{"files": rel, "items": entries}, func(w io.Writer) error {
				hint := false
				for _, e := range entries {
					printEntry(w, e)
					if strings.Contains(e.Message, "[2002]") {
						hint = true
					}
				}
				if len(entries) == 0 {
					fmt.Fprintln(w, "Không có entry nào khớp.")
				}
				if hint {
					fmt.Fprintln(w, "→ Kiểm tra MySQL: foxkit tcp 127.0.0.1:3306 hoặc foxkit svc")
				}
				return nil
			})
		},
	}
	log.Flags().String("level", "", "Level tối thiểu (debug…emergency)")
	log.Flags().Int("last", 20, "Số entry mới nhất")
	log.Flags().String("since", "", "Trong vòng (30m, 2h, 1d)")
	log.Flags().String("grep", "", "Lọc theo nội dung")
	log.Flags().Bool("full", false, "Đủ stack trace")
	log.Flags().Bool("follow", false, "Theo dõi liên tục")

	key := &cobra.Command{
		Use:   "key [dir]",
		Short: "Sinh APP_KEY, có thể ghi vào .env",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			apply, _ := c.Flags().GetBool("apply")
			force, _ := c.Flags().GetBool("force")
			dir, err := laraDir(args)
			if err != nil {
				return err
			}
			k, err := lara.NewKey()
			if err != nil {
				return output.Internal(err)
			}
			file := ""
			if apply {
				file, err = lara.ApplyKey(dir, k, force)
				var pe *env.ParseError
				switch {
				case errors.Is(err, lara.ErrKeyExists):
					return output.Fail("LARA_KEY_EXISTS", "APP_KEY đã có giá trị trong .env", "Thêm --force nếu chắc chắn muốn đổi")
				case errors.Is(err, os.ErrNotExist):
					return output.Fail("ENV_FILE_NOT_FOUND", "Không thấy file .env", "Tạo bằng: copy .env.example .env")
				case errors.As(err, &pe):
					return output.Fail("ENV_PARSE_FAILED", ".env sai cú pháp: "+pe.Error(), fmt.Sprintf("Kiểm tra dòng %d trong .env", pe.Line))
				case err != nil:
					return output.Internal(err)
				}
			}
			return printer(c).Success(map[string]any{"key": k, "applied": apply, "file": file}, func(w io.Writer) error {
				fmt.Fprintln(w, k)
				if apply {
					fmt.Fprintf(w, "Đã ghi vào %s (backup: .env.foxkit.bak)\n", file)
				} else {
					fmt.Fprintln(w, output.DryRunNote)
				}
				return nil
			})
		},
	}
	key.Flags().Bool("apply", false, "Ghi vào .env")
	key.Flags().Bool("force", false, "Ghi đè APP_KEY cũ")

	clear := &cobra.Command{
		Use:   "clear [dir]",
		Short: "php artisan optimize:clear",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			dir, err := laraDir(args)
			if err != nil {
				return err
			}
			ctx, stop := cmdContext(c)
			defer stop()
			res, err := lara.Clear(ctx, dir)
			if errors.Is(err, lara.ErrPHPNotFound) {
				return output.Fail("LARA_PHP_NOT_FOUND", "Không tìm thấy php trong PATH", "Kiểm tra bằng foxkit which php")
			}
			if err != nil {
				return canceled(err)
			}
			perr := printer(c).Success(res, func(w io.Writer) error {
				_, err := io.WriteString(w, res.Output)
				return err
			})
			if perr != nil {
				return perr
			}
			if res.ExitCode != 0 {
				return output.Fail("LARA_ARTISAN_FAILED", fmt.Sprintf("artisan kết thúc với mã %d", res.ExitCode), "Xem output phía trên")
			}
			return nil
		},
	}
	parent.AddCommand(log, key, clear)
	rootCmd.AddCommand(parent)
}
