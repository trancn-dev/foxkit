package cmd

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"slices"
	"strings"

	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "foxkit",
	Short:         "Bộ công cụ CLI nhỏ cho Windows",
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "In kết quả dạng JSON")
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return output.Usage("ARGS_INVALID", "Tham số không hợp lệ: "+err.Error(), "Xem: "+c.CommandPath()+" --help")
	})
}

// printer lấy cờ --json và writer của cobra (để test bắt được output).
func printer(c *cobra.Command) output.Printer {
	j, _ := c.Flags().GetBool("json") // cờ persistent luôn tồn tại
	return output.Printer{JSON: j, Stdout: c.OutOrStdout(), Stderr: c.ErrOrStderr()}
}

// Execute chạy root với os.Args và trả exit code.
func Execute() int { return run(os.Args[1:]) }

func run(args []string) int {
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		// Khi cobra lỗi trước lúc parse xong cờ, --json chưa được đọc: tự dò trong args.
		j := slices.Contains(args, "--json")
		// Lỗi kiểm tra đối số của chính cobra là plain error → quy về exit 2.
		var oe *output.Error
		if !errors.As(err, &oe) {
			for _, p := range []string{"unknown command", "accepts ", "requires ", "invalid argument"} {
				if strings.HasPrefix(err.Error(), p) {
					err = output.Usage("ARGS_INVALID", "Tham số không hợp lệ: "+err.Error(), "Xem: foxkit --help")
					break
				}
			}
		}
		return output.Printer{JSON: j, Stdout: rootCmd.OutOrStdout(), Stderr: rootCmd.ErrOrStderr()}.Error(err)
	}
	return 0
}

// cmdContext: context hủy được bằng Ctrl+C (cho các lệnh quét lâu).
func cmdContext(c *cobra.Command) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(c.Context(), os.Interrupt)
}

// canceled quy lỗi context bị hủy về CANCELED; lỗi khác giữ nguyên.
func canceled(err error) error {
	if errors.Is(err, context.Canceled) {
		return output.Canceled()
	}
	return err
}
