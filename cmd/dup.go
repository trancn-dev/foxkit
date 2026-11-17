package cmd

import (
	"fmt"
	"io"
	"time"

	"foxkit/internal/dup"
	"foxkit/internal/fsutil"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "dup <dir>...",
		Short: "Tìm file trùng nội dung, báo dung lượng lãng phí",
		RunE: func(c *cobra.Command, dirs []string) error {
			minStr, _ := c.Flags().GetString("min-size")
			exts, _ := c.Flags().GetStringSlice("ext")
			workers, _ := c.Flags().GetInt("workers")
			hidden, _ := c.Flags().GetBool("include-hidden")
			if len(dirs) == 0 {
				return output.Usage("DUP_NO_DIR", "Thiếu thư mục cần quét", `Ví dụ: foxkit dup D:\Photos`)
			}
			for _, d := range dirs {
				if !fsutil.IsDir(d) {
					return output.DirNotFound(d)
				}
			}
			minSize, err := fsutil.ParseSize(minStr)
			if err != nil {
				return output.SizeInvalid(minStr)
			}
			if c.Flags().Changed("workers") && (workers < 1 || workers > 64) {
				return output.Usage("ARGS_INVALID", "--workers phải từ 1 đến 64", "Ví dụ: --workers 8")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			opt := dup.Options{Dirs: dirs, MinSize: minSize, Exts: exts, Workers: workers, IncludeHidden: hidden}
			p := printer(c)
			if !p.JSON && isTerminal(c.ErrOrStderr()) {
				last := time.Now()
				opt.Progress = func(n int) {
					if time.Since(last) > 500*time.Millisecond {
						last = time.Now()
						fmt.Fprintf(c.ErrOrStderr(), "\rĐã quét %s file...", fsutil.FormatInt(n))
					}
				}
			}
			rep, err := dup.Find(ctx, opt)
			if err != nil {
				return canceled(err)
			}
			return p.Success(rep, func(w io.Writer) error {
				fmt.Fprintf(w, "Quét %s file trong %.1fs. Tìm thấy %d nhóm trùng, lãng phí %s.\n", fsutil.FormatInt(rep.Scanned), float64(rep.DurationMs)/1000, len(rep.Groups), fsutil.FormatBytes(rep.WastedBytes))
				for _, g := range rep.Groups {
					fmt.Fprintf(w, "\n[%s × %d]\n", fsutil.FormatBytes(g.Size), len(g.Files))
					for _, f := range g.Files {
						fmt.Fprintf(w, "  %s\n", f)
					}
				}
				if rep.Skipped > 0 {
					fmt.Fprintf(w, "\n(%d file không đọc được)\n", rep.Skipped)
				}
				return nil
			})
		},
	}
	c.Flags().String("min-size", "1KB", "Bỏ file nhỏ hơn")
	c.Flags().StringSlice("ext", nil, "Chỉ các đuôi file (jpg,png)")
	c.Flags().Int("workers", 0, "Số luồng băm (1-64)")
	c.Flags().Bool("include-hidden", false, "Gồm file/thư mục ẩn")
	rootCmd.AddCommand(c)
}
