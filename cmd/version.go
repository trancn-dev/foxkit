package cmd

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// Gán lúc build bằng -ldflags "-X foxkit/cmd.version=..."
var (
	version = "dev"
	commit  = "none"
	built   = "unknown"
)

type versionInfo struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
	Go      string `json:"go"`
}

func currentVersion() versionInfo {
	v := versionInfo{version, commit, built, runtime.Version()}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch {
			case s.Key == "vcs.revision" && v.Commit == "none" && len(s.Value) >= 7:
				v.Commit = s.Value[:7]
			case s.Key == "vcs.time" && v.Built == "unknown":
				v.Built = s.Value
			}
		}
	}
	return v
}

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Phiên bản foxkit",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			v := currentVersion()
			return printer(c).Success(v, func(w io.Writer) error {
				_, err := fmt.Fprintf(w, "foxkit %s (%s, %s, %s)\n", v.Version, v.Commit, v.Built, v.Go)
				return err
			})
		},
	})
}
