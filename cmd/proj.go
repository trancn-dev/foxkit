package cmd

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"foxkit/internal/fsutil"
	"foxkit/internal/output"
	"foxkit/internal/proj"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "proj [dir]",
		Short: "Nhận diện Laravel/Vue/Node/Go, phiên bản, scripts",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			if !fsutil.IsDir(dir) {
				return output.DirNotFound(dir)
			}
			in, err := proj.Detect(dir)
			var fe *proj.FileError
			if errors.As(err, &fe) {
				return output.Fail("PROJ_FILE_INVALID", fe.Error(), "Kiểm tra bằng foxkit json "+fe.File+" --validate")
			}
			if err != nil {
				return output.Internal(err)
			}
			return printer(c).Success(in, func(w io.Writer) error {
				head := fmt.Sprintf("%s  (%s)", in.Name, in.Dir)
				if in.Git != nil {
					head += "  · branch " + in.Git.Branch
				}
				fmt.Fprintln(w, head)
				if len(in.Types) == 0 {
					_, err := fmt.Fprintln(w, "Không nhận diện được loại dự án nào.")
					return err
				}
				var kinds []string
				if in.PHP != nil && in.PHP.Framework != "" {
					kinds = append(kinds, strings.TrimSpace("Laravel "+in.PHP.Version))
				}
				if in.Node != nil && in.Node.Framework != "" {
					k := in.Node.Framework + " " + in.Node.FrameworkVer
					if in.Node.Bundler != "" {
						k += " (" + in.Node.Bundler + ")"
					}
					kinds = append(kinds, strings.TrimSpace(k))
				}
				if in.Go != nil {
					kinds = append(kinds, "Go "+in.Go.GoVersion)
				}
				if len(kinds) == 0 {
					kinds = in.Types
				}
				fmt.Fprintf(w, "Loại     %s\n", strings.Join(kinds, " + "))
				if in.PHP != nil {
					pk := make([]string, 0, len(in.PHP.Packages))
					for k, v := range in.PHP.Packages {
						pk = append(pk, k+" "+v)
					}
					sort.Strings(pk)
					fmt.Fprintf(w, "PHP      %s  · %s\n", in.PHP.Require, strings.Join(pk, ", "))
				}
				if in.Node != nil {
					fmt.Fprintf(w, "Node     %s · engines %s · .nvmrc %s\n", in.Node.PackageManager, in.Node.Engines, in.Node.Nvmrc)
				}
				if len(in.Scripts) > 0 {
					fmt.Fprintln(w, "\nScripts")
					for _, s := range in.Scripts {
						fmt.Fprintf(w, "  %-22s %s\n", s.Run, s.Command)
					}
				}
				if len(in.Warnings) > 0 {
					fmt.Fprintln(w, "Cảnh báo")
					for _, wn := range in.Warnings {
						fmt.Fprintf(w, "  ! %s\n", wn)
					}
				}
				return nil
			})
		},
	}
	rootCmd.AddCommand(c)
}
