package cmd

import (
	"errors"
	"fmt"
	"io"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/syncdir"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "sync <src> <dst>",
		Short: "Sao lưu một chiều A → B, chỉ chép file mới/đổi",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			opt := syncdir.Options{}
			opt.Delete, _ = f.GetBool("delete")
			opt.Exclude, _ = f.GetStringSlice("exclude")
			opt.NoDefaultExcludes, _ = f.GetBool("no-default-excludes")
			opt.Compare, _ = f.GetString("compare")
			apply, _ := f.GetBool("apply")
			if opt.Compare != "size" && opt.Compare != "hash" {
				return output.Usage("ARGS_INVALID", "--compare phải là size hoặc hash", "")
			}
			if !fsutil.IsDir(args[0]) {
				return output.DirNotFound(args[0])
			}
			ctx, stop := cmdContext(c)
			defer stop()
			plan, err := syncdir.Build(ctx, args[0], args[1], opt)
			var full *syncdir.DiskFullError
			switch {
			case errors.Is(err, syncdir.ErrOverlap):
				return output.Usage("SYNC_OVERLAP", "Nguồn và đích trùng hoặc lồng nhau", "Nguồn và đích không được trùng hoặc lồng nhau")
			case errors.Is(err, syncdir.ErrDstUnsafe):
				return output.Usage("SYNC_DST_UNSAFE", "Đích không an toàn: "+plan.Dst, `Chọn một thư mục con riêng cho bản sao, ví dụ E:\Backup\Projects`)
			case errors.As(err, &full):
				return output.Fail("DISK_FULL", fmt.Sprintf("Đích không đủ chỗ: cần %s, còn %s", fsutil.FormatBytes(full.Need), fsutil.FormatBytes(full.Have)), "Giải phóng dung lượng hoặc chọn ổ khác")
			case err != nil:
				return canceled(err)
			}

			p := printer(c)
			tty := apply && !p.JSON && isTerminal(c.ErrOrStderr())
			if apply {
				emit := func(e syncdir.Event) {
					switch {
					case p.JSON:
						_ = p.Event(e) // pipe đóng thì lần chép cũng sắp dừng
					case tty:
						fmt.Fprintf(c.ErrOrStderr(), "\r%d/%d file · %s   ", e.Done, e.Total, fsutil.FormatBytes(e.Bytes))
					}
				}
				if err := syncdir.Apply(ctx, &plan, opt, emit); err != nil {
					return canceled(err)
				}
				if tty {
					fmt.Fprintln(c.ErrOrStderr())
				}
			}
			var nCopy, nUpdate, nDel int
			var bCopy, bUpdate int64
			for _, a := range plan.Actions {
				switch a.Op {
				case "copy":
					nCopy++
					bCopy += a.Bytes
				case "update":
					nUpdate++
					bUpdate += a.Bytes
				case "delete":
					nDel++
				}
			}
			var perr error
			if !p.JSON || !apply { // --json + --apply đã in NDJSON ở trên; chỉ in envelope khi xem trước
				perr = p.Success(plan, func(w io.Writer) error {
					fmt.Fprintf(w, "Nguồn  %s        Đích  %s\n", plan.Src, plan.Dst)
					verb := ""
					if plan.Applied {
						verb = "đã "
					}
					fmt.Fprintf(w, "  + %s%d file mới (%s)\n  ~ %s%d file đã đổi (%s)\n  = %d file giống nhau\n", verb, nCopy, fsutil.FormatBytes(bCopy), verb, nUpdate, fsutil.FormatBytes(bUpdate), plan.Same)
					switch {
					case opt.Delete:
						fmt.Fprintf(w, "  − %s%d mục chỉ có ở đích (chuyển vào .foxkit-trash)\n", verb, nDel)
					case plan.OnlyInDst > 0:
						fmt.Fprintf(w, "  − %d mục chỉ có ở đích (giữ nguyên; dùng --delete để dọn)\n", plan.OnlyInDst)
					}
					for _, fl := range plan.Failed {
						fmt.Fprintf(w, "  ! %s: %s\n", fl.Path, fl.Reason)
					}
					if !apply {
						fmt.Fprintln(w, output.DryRunNote)
					}
					return nil
				})
			}
			if perr != nil {
				return perr
			}
			if len(plan.Failed) > 0 {
				return output.Fail("SYNC_PARTIAL", fmt.Sprintf("%d file không chép được", len(plan.Failed)), "Xem mục failed; đóng chương trình đang dùng file rồi chạy lại")
			}
			return nil
		},
	}
	f := c.Flags()
	f.Bool("delete", false, "Chuyển file thừa ở đích vào .foxkit-trash")
	f.StringSlice("exclude", nil, "Glob loại thêm")
	f.Bool("no-default-excludes", false, "Tắt danh sách loại mặc định")
	f.String("compare", "size", "size|hash")
	f.Bool("apply", false, "Thực hiện")
	rootCmd.AddCommand(c)
}
