package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/temp"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "temp",
		Short: "Xem và dọn file tạm cũ của user",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			olderStr, _ := c.Flags().GetString("older")
			apply, _ := c.Flags().GetBool("apply")
			older, err := fsutil.ParseAge(olderStr)
			if err != nil || older < time.Hour {
				return output.AgeInvalid(olderStr)
			}
			dir := os.TempDir()
			if !fsutil.IsDir(dir) {
				return output.Fail("TEMP_NOT_FOUND", "Không tìm thấy thư mục tạm "+dir, "Kiểm tra biến môi trường TEMP")
			}
			ctx, stop := cmdContext(c)
			defer stop()
			rep, err := temp.Run(ctx, dir, older, apply, time.Now())
			if err != nil {
				return canceled(err)
			}
			return printer(c).Success(rep, func(w io.Writer) error {
				fmt.Fprintf(w, "Thư mục tạm: %s  (%s)\n", rep.Dir, fsutil.FormatBytes(rep.TotalBytes))
				if !apply {
					fmt.Fprintf(w, "Có thể dọn %s mục cũ hơn %s: %s\n%s\n", fsutil.FormatInt(rep.Candidates), olderStr, fsutil.FormatBytes(rep.CandidateBytes), output.DryRunNote)
					return nil
				}
				_, err := fmt.Fprintf(w, "Đã xóa %s mục, giải phóng %s. Bỏ qua %d mục đang được sử dụng.\n", fsutil.FormatInt(rep.Deleted), fsutil.FormatBytes(rep.FreedBytes), rep.Locked)
				return err
			})
		},
	}
	c.Flags().String("older", "7d", "Chỉ xóa mục cũ hơn mốc này (tối thiểu 1h)")
	c.Flags().Bool("apply", false, "Xóa thật")
	rootCmd.AddCommand(c)
}
