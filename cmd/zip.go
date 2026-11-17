package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/zipx"

	"github.com/spf13/cobra"
)

func zipError(err error) error {
	switch {
	case errors.Is(err, zipx.ErrNotFound):
		return output.Usage("ZIP_NOT_FOUND", err.Error(), "Kiểm tra lại đường dẫn")
	case errors.Is(err, zipx.ErrInvalid):
		return output.Fail("ZIP_INVALID", err.Error(), "File có thể bị tải lỗi; kiểm tra bằng foxkit hash")
	case errors.Is(err, zipx.ErrUnsafePath):
		return output.Fail("ZIP_UNSAFE_PATH", err.Error(), "File zip chứa đường dẫn nguy hiểm, không giải nén")
	case errors.Is(err, zipx.ErrTooLarge):
		return output.Fail("ZIP_TOO_LARGE", err.Error(), "Giới hạn: 20 lần dung lượng zip hoặc 20GB; dùng 7-Zip nếu file đáng tin")
	case errors.Is(err, zipx.ErrExists):
		return output.Fail("FILE_EXISTS", err.Error(), "Thêm --force để ghi đè")
	}
	return canceled(err)
}

func init() {
	parent := &cobra.Command{Use: "zip", Short: "Nén và giải nén ZIP"}

	pack := &cobra.Command{
		Use:   "pack <src>...",
		Short: "Nén thư mục/file, tự bỏ node_modules, .git, .env",
		RunE: func(c *cobra.Command, srcs []string) error {
			f := c.Flags()
			out, _ := f.GetString("out")
			exclude, _ := f.GetStringSlice("exclude")
			noDef, _ := f.GetBool("no-default-excludes")
			level, _ := f.GetInt("level")
			force, _ := f.GetBool("force")
			if len(srcs) == 0 {
				return output.Usage("ZIP_NO_SOURCE", "Thiếu thư mục/file cần nén", `Ví dụ: foxkit zip pack D:\Projects\shop`)
			}
			if level < 0 || level > 9 {
				return output.Usage("ARGS_INVALID", "--level phải từ 0 đến 9", "")
			}
			for _, s := range srcs {
				if !fsutil.Exists(s) {
					return output.Usage("ZIP_NOT_FOUND", "Không tìm thấy "+s, "Kiểm tra lại đường dẫn")
				}
			}
			if out == "" {
				if len(srcs) > 1 {
					return output.Usage("ARGS_INVALID", "Nhiều nguồn thì bắt buộc chỉ định --out", "Ví dụ: --out backup.zip")
				}
				out = zipx.DefaultOut(srcs[0])
			}
			ctx, stop := cmdContext(c)
			defer stop()
			res, err := zipx.Pack(ctx, srcs, out, zipx.PackOptions{Exclude: exclude, NoDefaultExcludes: noDef, Level: level, Force: force})
			if err != nil {
				return zipError(err)
			}
			return printer(c).Success(res, func(w io.Writer) error {
				ratio := 0.0
				if res.Bytes > 0 {
					ratio = 100 * (1 - float64(res.ZipBytes)/float64(res.Bytes))
				}
				fmt.Fprintf(w, "Nén %s file (%s) → %s (%s, nén %.0f%%)\n", fsutil.FormatInt(res.Files), fsutil.FormatBytes(res.Bytes), res.Out, fsutil.FormatBytes(res.ZipBytes), ratio)
				if len(res.Excluded) > 0 {
					fmt.Fprintf(w, "Đã loại: %s\n", strings.Join(res.Excluded, ", "))
				}
				if len(res.Skipped) > 0 {
					fmt.Fprintf(w, "Bỏ qua (không đọc được/symlink): %s\n", strings.Join(res.Skipped, ", "))
				}
				return nil
			})
		},
	}
	pack.Flags().String("out", "", "File zip đầu ra")
	pack.Flags().StringSlice("exclude", nil, "Loại thêm (glob tên hoặc đường dẫn)")
	pack.Flags().Bool("no-default-excludes", false, "Không tự loại .git/node_modules/vendor/.env/*.log")
	pack.Flags().Int("level", 6, "Mức nén 0-9")
	pack.Flags().Bool("force", false, "Ghi đè")

	unpack := &cobra.Command{
		Use:   "unpack <file.zip>",
		Short: "Giải nén file zip an toàn",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			to, _ := c.Flags().GetString("to")
			force, _ := c.Flags().GetBool("force")
			if !fsutil.Exists(args[0]) {
				return output.Usage("ZIP_NOT_FOUND", "Không tìm thấy "+args[0], "Kiểm tra lại đường dẫn")
			}
			if to == "" {
				to = zipx.DefaultTo(args[0])
			}
			ctx, stop := cmdContext(c)
			defer stop()
			res, err := zipx.Unpack(ctx, args[0], to, force)
			if err != nil {
				return zipError(err)
			}
			return printer(c).Success(res, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "Đã giải nén %s file (%s) vào %s\n", fsutil.FormatInt(res.Files), fsutil.FormatBytes(res.Bytes), res.To)
				return err
			})
		},
	}
	unpack.Flags().String("to", "", "Thư mục giải nén")
	unpack.Flags().Bool("force", false, "Ghi đè file đã có")

	list := &cobra.Command{
		Use:   "list <file.zip>",
		Short: "Liệt kê file trong zip",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			entries, err := zipx.List(args[0])
			if err != nil {
				return zipError(err)
			}
			return printer(c).Success(map[string]any{"items": entries}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "TÊN\tSIZE\tNÉN\tNGÀY")
				for _, e := range entries {
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.Name, fsutil.FormatBytes(e.Bytes), fsutil.FormatBytes(e.Compressed), e.Modified.Format("2006-01-02 15:04"))
				}
				return tw.Flush()
			})
		},
	}
	parent.AddCommand(pack, unpack, list)
	rootCmd.AddCommand(parent)
}
