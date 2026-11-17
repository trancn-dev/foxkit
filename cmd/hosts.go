package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"foxkit/internal/hosts"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func loadHosts(path string) (*hosts.File, error) {
	f, err := hosts.Load(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return nil, output.Fail("HOSTS_NOT_FOUND", "Không thấy file hosts "+path, "Kiểm tra biến FOXKIT_HOSTS hoặc quyền đọc")
	case errors.Is(err, os.ErrPermission):
		return nil, output.NeedAdmin("Không đọc được file hosts")
	case err != nil:
		return nil, output.Internal(err)
	}
	return f, nil
}

var symbols = map[string]string{"add": "+", "remove": "-", "enable": "~", "disable": "x", "skip": "="}
var notes = map[string]string{"add": "thêm", "remove": "xóa", "enable": "bật lại", "disable": "tắt", "skip": "(không đổi, bỏ qua)"}

// runHostsEdit: tải file → sửa trong bộ nhớ → (nếu --apply) ghi. mutate trả danh sách thay đổi.
func runHostsEdit(c *cobra.Command, domains []string, mutate func(f *hosts.File) []hosts.Change) error {
	apply, _ := c.Flags().GetBool("apply")
	if len(domains) == 0 {
		return output.Usage("ARGS_INVALID", "Thiếu domain", "Ví dụ: foxkit hosts add shop.test")
	}
	for _, d := range domains {
		if !hosts.ValidDomain(d) {
			return output.Usage("HOSTS_DOMAIN_INVALID", "Domain không hợp lệ: "+d, "Ví dụ: shop.test")
		}
	}
	path := hosts.DefaultPath()
	f, err := loadHosts(path)
	if err != nil {
		return err
	}
	changes := mutate(f)
	for _, ch := range changes {
		switch ch.Action {
		case "conflict":
			return output.Fail("HOSTS_DOMAIN_EXISTS", fmt.Sprintf("%s đã có trong hosts với IP %s", ch.Domain, ch.IP), "Xóa trước: foxkit hosts rm "+ch.Domain+" --apply")
		case "notfound":
			return output.Fail("HOSTS_NOT_FOUND", ch.Domain+" không có trong hosts", "Xem danh sách: foxkit hosts list")
		}
	}
	changed := false
	for _, ch := range changes {
		if ch.Action != "skip" {
			changed = true
		}
	}
	backup := ""
	if apply && changed {
		if backup, err = hosts.Save(path, f); err != nil {
			if errors.Is(err, os.ErrPermission) {
				e := output.NeedAdmin("Không ghi được file hosts")
				e.Hint = "Chạy terminal bằng quyền Administrator. Nếu vẫn lỗi, phần mềm diệt virus có thể đang khóa file hosts"
				return e
			}
			return output.Fail("HOSTS_WRITE_FAILED", "Ghi file hosts lỗi: "+err.Error(), "Kiểm tra quyền ghi và phần mềm diệt virus")
		}
	}
	data := map[string]any{"applied": apply && changed, "changes": changes, "backup": backup, "file": path}
	return printer(c).Success(data, func(w io.Writer) error {
		for _, ch := range changes {
			fmt.Fprintf(w, "%s %-12s %s  %s\n", symbols[ch.Action], ch.IP, ch.Domain, notes[ch.Action])
		}
		switch {
		case !apply:
			fmt.Fprintln(w, output.DryRunNote)
		case changed:
			fmt.Fprintf(w, "Đã ghi %s (backup: %s).\nTrình duyệt có thể cache DNS, chạy `foxkit dns --flush` nếu chưa thấy thay đổi.\n", path, backup)
		}
		return nil
	})
}

func init() {
	parent := &cobra.Command{Use: "hosts", Short: "Quản lý file hosts"}

	list := &cobra.Command{
		Use:   "list",
		Short: "Liệt kê domain trong file hosts",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			managed, _ := c.Flags().GetBool("managed")
			path := hosts.DefaultPath()
			f, err := loadHosts(path)
			if err != nil {
				return err
			}
			entries := []hosts.Entry{}
			for _, e := range f.Entries() {
				if !managed || e.Managed {
					entries = append(entries, e)
				}
			}
			return printer(c).Success(map[string]any{"file": path, "entries": entries}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "IP\tDOMAIN\tTRẠNG THÁI")
				for _, e := range entries {
					st := "bật"
					if !e.Enabled {
						st = "tắt"
					}
					if e.Managed {
						st += "   (foxkit)"
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\n", e.IP, e.Domain, st)
				}
				return tw.Flush()
			})
		},
	}
	list.Flags().Bool("managed", false, "Chỉ dòng foxkit thêm")

	add := &cobra.Command{
		Use:   "add <domain>...",
		Short: "Thêm domain vào file hosts",
		RunE: func(c *cobra.Command, domains []string) error {
			ip, _ := c.Flags().GetString("ip")
			if !hosts.ValidIP(ip) {
				return output.Usage("HOSTS_IP_INVALID", "IP không hợp lệ: "+ip, "Ví dụ: --ip 127.0.0.1")
			}
			return runHostsEdit(c, domains, func(f *hosts.File) []hosts.Change { return f.Add(ip, domains) })
		},
	}
	add.Flags().String("ip", "127.0.0.1", "IP")

	rm := &cobra.Command{
		Use:   "rm <domain>...",
		Short: "Xóa domain khỏi file hosts",
		RunE: func(c *cobra.Command, domains []string) error {
			return runHostsEdit(c, domains, func(f *hosts.File) []hosts.Change { return f.Remove(domains) })
		},
	}
	toggle := func(use, short string, on bool) *cobra.Command {
		return &cobra.Command{
			Use:   use + " <domain>...",
			Short: short,
			RunE: func(c *cobra.Command, domains []string) error {
				return runHostsEdit(c, domains, func(f *hosts.File) []hosts.Change { return f.SetEnabled(domains, on) })
			},
		}
	}
	on, off := toggle("on", "Bật lại domain đã tắt (bỏ comment)", true), toggle("off", "Tắt domain (comment dòng lại)", false)
	for _, sc := range []*cobra.Command{add, rm, on, off} {
		sc.Flags().Bool("apply", false, "Ghi file (cần admin)")
	}
	parent.AddCommand(list, add, rm, on, off)
	rootCmd.AddCommand(parent)
}
