package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"foxkit/internal/output"
	"foxkit/internal/startup"
	"foxkit/internal/svc"

	"github.com/spf13/cobra"
)

func svcError(err error) error {
	switch {
	case errors.Is(err, svc.ErrAccess):
		return output.NeedAdmin("Không đủ quyền thao tác service")
	case errors.Is(err, svc.ErrNotFound):
		return output.Fail("SVC_NOT_FOUND", err.Error(), "Xem danh sách: foxkit svc --all")
	case errors.Is(err, svc.ErrTimeout):
		return output.Fail("SVC_TIMEOUT", "Service chưa đổi trạng thái sau 30 giây", "Xem Event Viewer hoặc log của service")
	case errors.Is(err, svc.ErrDependents):
		return output.Fail("SVC_HAS_DEPENDENTS", "Còn service phụ thuộc đang chạy", "Dừng các service phụ thuộc trước")
	case errors.Is(err, svc.ErrDisabled):
		return output.Fail("SVC_DISABLED", "Service đang bị vô hiệu hóa", "Đổi Startup type trong services.msc")
	case errors.Is(err, svc.ErrUnsupported):
		return output.Fail("SVC_UNSUPPORTED", "Chỉ hỗ trợ Windows", "")
	}
	return output.Fail("SVC_FAILED", err.Error(), "")
}

func init() {
	st := &cobra.Command{
		Use:   "startup",
		Short: "App tự chạy khi đăng nhập Windows",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			items, warns, err := startup.List()
			if err != nil {
				return output.Fail("STARTUP_FAILED", "Không đọc được chương trình khởi động: "+err.Error(), "")
			}
			return printer(c).Success(map[string]any{"items": items}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "TRẠNG THÁI\tTÊN\tNGUỒN\tLỆNH")
				for _, it := range items {
					state := "bật"
					if !it.Enabled {
						state = "tắt"
					}
					note := ""
					if !it.Exists {
						state += " ✗"
						note = "   (file không còn)"
					}
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s%s\n", state, it.Name, it.Source, it.Command, note)
				}
				if err := tw.Flush(); err != nil {
					return err
				}
				for _, wn := range warns {
					fmt.Fprintf(c.ErrOrStderr(), "Cảnh báo: %s\n", wn)
				}
				_, err := fmt.Fprintln(w, "Tắt bớt: Task Manager → Startup apps.")
				return err
			})
		},
	}
	rootCmd.AddCommand(st)

	list := &cobra.Command{
		Use:   "svc [filter]",
		Short: "Xem service dev: MySQL, Redis, Apache...",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			all, _ := c.Flags().GetBool("all")
			filter := ""
			if len(args) == 1 {
				filter = args[0]
			}
			items, err := svc.List(filter, all)
			if err != nil {
				return svcError(err)
			}
			return printer(c).Success(map[string]any{"items": items}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, "TRẠNG THÁI\tTÊN\tHIỂN THỊ\tKHỞI ĐỘNG")
				for _, s := range items {
					fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.State, s.Name, s.Display, s.StartType)
				}
				return tw.Flush()
			})
		},
	}
	list.Flags().Bool("all", false, "Mọi service")

	action := func(verb, short string, f func(string, time.Duration) (svc.Service, error)) *cobra.Command {
		return &cobra.Command{
			Use:   verb + " <name>",
			Short: short,
			Args:  cobra.ExactArgs(1),
			RunE: func(c *cobra.Command, args []string) error {
				s, err := f(args[0], 30*time.Second)
				if err != nil {
					return svcError(err)
				}
				return printer(c).Success(map[string]any{"service": s, "changed": true}, func(w io.Writer) error {
					pid := ""
					if s.PID != 0 {
						pid = fmt.Sprintf(" (PID %d)", s.PID)
					}
					_, err := fmt.Fprintf(w, "%s: %s%s\n", s.Name, strings.ToLower(s.State), pid)
					return err
				})
			},
		}
	}
	list.AddCommand(
		action("start", "Start một Windows service", svc.Start),
		action("stop", "Stop một Windows service", svc.Stop),
		action("restart", "Restart một Windows service", svc.Restart),
	)
	rootCmd.AddCommand(list)
}
