package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/winapi"
	"foxkit/internal/winpath"

	"github.com/spf13/cobra"
)

func printPath(w io.Writer, r winpath.Report, entries []winpath.Entry) error {
	first := make(map[string]int)
	for _, e := range r.Entries {
		if _, ok := first[winpath.Normalize(e.Expanded)]; !ok && !e.Empty {
			first[winpath.Normalize(e.Expanded)] = e.Index
		}
	}
	for _, e := range entries {
		mark, note := "✓", ""
		switch {
		case e.Empty:
			mark, note = "✗", "(mục rỗng)"
		case !e.Exists:
			mark, note = "✗", "(không tồn tại)"
		case e.Duplicate:
			mark, note = "⚠", fmt.Sprintf("(trùng mục %d)", first[winpath.Normalize(e.Expanded)])
		}
		scope := ""
		if e.Scope != "" {
			scope = "[" + e.Scope + "] "
		}
		fmt.Fprintf(w, "%3d  %s  %s%-60s %s\n", e.Index, mark, scope, e.Dir, note)
	}
	warn := ""
	if r.Length > winpath.WarnLength {
		warn = " (dài quá 2047, một số công cụ cũ sẽ cắt)"
	}
	_, err := fmt.Fprintf(w, "PATH dài %s ký tự%s · %d vấn đề\n", fsutil.FormatInt(r.Length), warn, r.Problems)
	return err
}

// scoped trả báo cáo theo --scope: all = PATH của process gắn nhãn system/user/process; user/system = giá trị trong registry.
func scoped(scope string) (winpath.Report, error) {
	switch scope {
	case "all":
		r := winpath.Analyze(os.Getenv("PATH"))
		u, uerr := winpath.Read(winpath.User)
		s, serr := winpath.Read(winpath.System)
		if uerr == nil && serr == nil { // không đọc được registry thì thôi nhãn, vẫn hiện PATH
			r.Label(u.Value, s.Value)
		}
		return r, nil
	case "user", "system":
		raw, err := winpath.Read(winpath.Scope(scope))
		if err != nil {
			return winpath.Report{}, output.Fail("REGISTRY_FAILED", "Không đọc được PATH "+scope+": "+err.Error(), "")
		}
		return winpath.Analyze(raw.Value), nil
	}
	return winpath.Report{}, output.Usage("ARGS_INVALID", "--scope phải là all, user hoặc system", "")
}

// editPath dùng chung cho add/rm/clean: đọc registry → sửa trong bộ nhớ → (nếu --apply) backup + ghi + broadcast.
func editPath(c *cobra.Command, edit func(scope winpath.Scope, raw winpath.Raw) (string, []string, []string, error)) error {
	apply, _ := c.Flags().GetBool("apply")
	system, _ := c.Flags().GetBool("system")
	scope := winpath.User
	if system {
		scope = winpath.System
		if apply && !winapi.IsAdmin() { // kiểm trước khi ghi bất cứ thứ gì
			return output.NeedAdmin("Sửa PATH hệ thống cần quyền Administrator")
		}
	}
	raw, err := winpath.Read(scope)
	if err != nil {
		return output.Fail("REGISTRY_FAILED", "Không đọc được PATH "+string(scope)+": "+err.Error(), "")
	}
	newValue, added, removed, err := edit(scope, raw)
	if err != nil {
		return err
	}
	changed := newValue != raw.Value
	if len(newValue) > winpath.MaxLength {
		return output.Fail("PATH_TOO_LONG", fmt.Sprintf("PATH sẽ dài %d ký tự (tối đa %d)", len(newValue), winpath.MaxLength), "Dọn bớt bằng foxkit path clean")
	}
	backup := ""
	if apply && changed {
		if backup, err = winpath.Backup(scope, raw); err != nil {
			return output.Fail("REGISTRY_FAILED", "Không ghi được backup: "+err.Error(), "")
		}
		if err := winpath.Write(scope, winpath.Raw{Value: newValue, Expand: raw.Expand}); err != nil {
			if errors.Is(err, os.ErrPermission) {
				return output.NeedAdmin("Không ghi được PATH " + string(scope))
			}
			return output.Fail("REGISTRY_FAILED", "Ghi registry lỗi: "+err.Error(), "")
		}
		_ = winpath.Broadcast() // app chưa nhận được thông báo thì chỉ cần mở lại, không phải lỗi
	}
	if added == nil {
		added = []string{}
	}
	if removed == nil {
		removed = []string{}
	}
	data := map[string]any{"scope": string(scope), "applied": apply && changed, "added": added, "removed": removed, "backup": backup, "length": len(newValue)}
	return printer(c).Success(data, func(w io.Writer) error {
		if !changed {
			_, err := fmt.Fprintf(w, "PATH (%s) không có gì để đổi.\n", scope)
			return err
		}
		for _, a := range added {
			fmt.Fprintf(w, "  + %s\n", a)
		}
		for _, r := range removed {
			fmt.Fprintf(w, "  - %s\n", r)
		}
		if len(newValue) > winpath.WarnLength {
			fmt.Fprintf(w, "Cảnh báo: PATH dài %d ký tự, một số công cụ cũ cắt ở 2047.\n", len(newValue))
		}
		if !apply {
			fmt.Fprintln(w, output.DryRunNote)
			return nil
		}
		fmt.Fprintf(w, "Đã ghi PATH (%s). Backup: %s\nLưu ý: terminal/VS Code đang mở cần mở lại mới thấy PATH mới.\n", scope, backup)
		return nil
	})
}

func init() {
	path := &cobra.Command{
		Use:   "path",
		Short: "Liệt kê PATH dạng danh sách",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			scope, _ := c.Flags().GetString("scope")
			r, err := scoped(scope)
			if err != nil {
				return err
			}
			return printer(c).Success(r, func(w io.Writer) error {
				if err := printPath(w, r, r.Entries); err != nil {
					return err
				}
				_, err := fmt.Fprintln(w, "Sửa vĩnh viễn: foxkit path add|rm|clean --apply, hoặc Settings → Environment Variables.")
				return err
			})
		},
	}
	path.Flags().String("scope", "all", "all|user|system")

	check := &cobra.Command{
		Use:   "check",
		Short: "Tìm mục PATH không tồn tại, trùng, rỗng",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			r := winpath.Analyze(os.Getenv("PATH"))
			bad := r.Problematic()
			data := r
			data.Entries = bad
			return printer(c).Success(data, func(w io.Writer) error { return printPath(w, r, bad) })
		},
	}

	add := &cobra.Command{
		Use:   "add <dir>",
		Short: "Thêm thư mục vào PATH vĩnh viễn",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			first, _ := c.Flags().GetBool("first")
			abs, err := filepath.Abs(args[0])
			if err != nil {
				return output.Internal(err)
			}
			if !fsutil.IsDir(abs) {
				return output.Fail("PATH_DIR_NOT_FOUND", "Không tìm thấy thư mục "+abs, "Kiểm tra lại, hoặc tạo thư mục trước")
			}
			return editPath(c, func(scope winpath.Scope, raw winpath.Raw) (string, []string, []string, error) {
				dir := abs
				if scope == winpath.User {
					dir = winpath.Portable(abs)
				}
				v, changed := winpath.Add(raw.Value, dir, first)
				if !changed {
					return raw.Value, nil, nil, nil
				}
				return v, []string{dir}, nil, nil
			})
		},
	}
	add.Flags().Bool("first", false, "Thêm lên đầu (ưu tiên cao nhất)")

	rm := &cobra.Command{
		Use:   "rm <dir>",
		Short: "Xóa thư mục khỏi PATH vĩnh viễn",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return editPath(c, func(_ winpath.Scope, raw winpath.Raw) (string, []string, []string, error) {
				v, n := winpath.Remove(raw.Value, args[0])
				if n == 0 {
					return "", nil, nil, output.Fail("PATH_NOT_IN_PATH", args[0]+" không có trong PATH", "Xem danh sách: foxkit path --scope user")
				}
				return v, nil, []string{args[0]}, nil
			})
		},
	}

	clean := &cobra.Command{
		Use:   "clean",
		Short: "Xóa mục PATH không tồn tại và trùng",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return editPath(c, func(_ winpath.Scope, raw winpath.Raw) (string, []string, []string, error) {
				v, removed := winpath.Clean(raw.Value, func(p string) bool {
					_, err := os.Stat(p)
					return err == nil || !os.IsNotExist(err)
				})
				return v, nil, removed, nil
			})
		},
	}
	for _, sc := range []*cobra.Command{add, rm, clean} {
		sc.Flags().Bool("system", false, "PATH hệ thống (cần admin)")
		sc.Flags().Bool("apply", false, "Ghi registry")
	}
	path.AddCommand(check, add, rm, clean)
	rootCmd.AddCommand(path)
}
