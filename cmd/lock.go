package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"foxkit/internal/fsutil"
	"foxkit/internal/kill"
	"foxkit/internal/lock"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "lock <path>...",
		Short: "Tìm process đang khóa file/thư mục",
		RunE: func(c *cobra.Command, paths []string) error {
			doKill, _ := c.Flags().GetBool("kill")
			maxFiles, _ := c.Flags().GetInt("max-files")
			if len(paths) == 0 {
				return output.Usage("ARGS_INVALID", "Thiếu đường dẫn file hoặc thư mục", `Ví dụ: foxkit lock D:\tmp\demo\node_modules`)
			}
			if maxFiles < 1 || maxFiles > 50000 {
				return output.Usage("MAX_FILES_INVALID", "--max-files phải từ 1 đến 50000", "Ví dụ: --max-files 5000")
			}
			for _, p := range paths {
				if !fsutil.Exists(p) {
					return output.PathNotFound(p)
				}
			}
			res, err := lock.Find(paths, maxFiles)
			if errors.Is(err, os.ErrNotExist) {
				return output.PathNotFound(paths[0])
			}
			if err != nil {
				return output.Fail("RM_FAILED", "Restart Manager lỗi: "+err.Error(), "Kèm mã lỗi Windows ở trên")
			}
			var failed *output.Error
			if doKill && len(res.Holders) > 0 {
				var targets []kill.Target
				for _, h := range res.Holders {
					if h.Type == "explorer" || h.Type == "critical" {
						continue // Explorer chỉ gợi ý restart, không kill
					}
					targets = append(targets, kill.Target{PID: int(h.PID), Name: h.Name})
				}
				kr := kill.Kill(c.Context(), targets, false)
				for _, t := range kr.Killed {
					res.Killed = append(res.Killed, t.PID)
				}
				switch kill.ExitCode(kr) {
				case "NEED_ADMIN":
					failed = output.NeedAdmin("Không đủ quyền kill một số process đang giữ file")
				case "KILL_PARTIAL":
					failed = output.Fail("KILL_PARTIAL", fmt.Sprintf("%d process không kill được", len(kr.Failed)), "Xem danh sách bằng foxkit ps")
				}
			}
			perr := printer(c).Success(res, func(w io.Writer) error {
				fmt.Fprintf(w, "%v  (đã kiểm %s file)\n", paths, fsutil.FormatInt(res.Checked))
				if res.Truncated {
					fmt.Fprintf(w, "Thư mục có nhiều file hơn --max-files (%d), kết quả có thể chưa đủ.\n", maxFiles)
				}
				if len(res.Holders) == 0 {
					_, err := fmt.Fprintln(w, "Không có process nào đang giữ. Nếu vẫn không xóa được: kiểm tra quyền, thuộc tính read-only, hoặc phần mềm diệt virus đang quét.")
					return err
				}
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "PID\tTIẾN TRÌNH\tỨNG DỤNG\tLOẠI")
				for _, h := range res.Holders {
					fmt.Fprintf(tw, "%d\t%s\t%s\t%s\n", h.PID, h.Name, h.App, h.Type)
				}
				if err := tw.Flush(); err != nil {
					return err
				}
				if len(res.Killed) > 0 {
					fmt.Fprintf(w, "Đã kill: %v\n", res.Killed)
				} else {
					fmt.Fprintln(w, "Gợi ý: đóng các ứng dụng trên, hoặc chạy lại với --kill.")
				}
				return nil
			})
			if perr != nil {
				return perr
			}
			if failed != nil {
				return failed
			}
			return nil
		},
	}
	c.Flags().Bool("kill", false, "Kill các process này")
	c.Flags().Int("max-files", 5000, "Số file tối đa khi quét thư mục")
	rootCmd.AddCommand(c)
}
