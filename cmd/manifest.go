package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"foxkit/internal/manifest"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "manifest",
		Short: "Danh sách lệnh cho launcher",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			return printer(c).Success(json.RawMessage(manifest.Raw), func(w io.Writer) error {
				f, err := manifest.Parse()
				if err != nil {
					return err
				}
				last := ""
				for _, cm := range f.Commands {
					if cm.Category != last {
						fmt.Fprintf(w, "[%s]\n", cm.Category)
						last = cm.Category
					}
					fmt.Fprintf(w, "  %-14s —  %s\n", cm.Name, cm.Description)
				}
				return nil
			})
		},
	})
}
