package cmd

import (
	"fmt"
	"io"
	"os"

	"foxkit/internal/fsutil"
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
		fmt.Fprintf(w, "%3d  %s  %-60s %s\n", e.Index, mark, e.Dir, note)
	}
	warn := ""
	if r.Length > winpath.LongPath {
		warn = " (dài quá 2047, một số công cụ cũ sẽ cắt)"
	}
	_, err := fmt.Fprintf(w, "PATH dài %s ký tự%s · %d vấn đề\n", fsutil.FormatInt(r.Length), warn, r.Problems)
	return err
}

func init() {
	path := &cobra.Command{
		Use:   "path",
		Short: "Liệt kê PATH dạng danh sách",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			r := winpath.Analyze(os.Getenv("PATH"))
			return printer(c).Success(r, func(w io.Writer) error {
				if err := printPath(w, r, r.Entries); err != nil {
					return err
				}
				_, err := fmt.Fprintln(w, "Sửa vĩnh viễn: foxkit path rm <dir> hoặc Settings → Environment Variables.")
				return err
			})
		},
	}
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
	path.AddCommand(check)
	rootCmd.AddCommand(path)
}
