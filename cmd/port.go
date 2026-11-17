package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"foxkit/internal/output"
	"foxkit/internal/port"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "port [port]",
		Short: "Xem process nào đang chiếm port, có thể kill",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runPort,
	}
	c.Flags().Bool("kill", false, "Kill process đang chiếm port")
	c.Flags().Bool("list", false, "Liệt kê mọi port TCP đang LISTEN")
	rootCmd.AddCommand(c)
}

func runPort(c *cobra.Command, args []string) error {
	kill, _ := c.Flags().GetBool("kill")
	list, _ := c.Flags().GetBool("list")
	if list && (len(args) > 0 || kill) {
		return output.Usage("ARGS_CONFLICT", "--list không dùng chung với port hoặc --kill", "Dùng: foxkit port --list")
	}
	netstatErr := func(err error) error {
		return output.Fail("NETSTAT_FAILED", "Không đọc được danh sách kết nối", `Thử chạy "netstat -ano" trong terminal để kiểm tra`)
	}
	ctx := c.Context()
	p := printer(c)

	if list {
		items, err := port.List(ctx)
		if err != nil {
			return netstatErr(err)
		}
		return p.Success(map[string]any{"items": items}, func(w io.Writer) error {
			tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "PORT\tPID\tPROCESS\tADDRESS")
			for _, l := range items {
				fmt.Fprintf(tw, "%d\t%d\t%s\t%s\n", l.Port, l.PID, l.Process, l.Address)
			}
			return tw.Flush()
		})
	}

	if len(args) == 0 {
		return output.Usage("PORT_MISSING", "Thiếu số port", "Ví dụ: foxkit port 8000")
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 1 || n > 65535 {
		return output.PortInvalid()
	}
	res, err := port.Find(ctx, n)
	if err != nil {
		return netstatErr(err)
	}

	if kill && res.InUse {
		seen := map[int]bool{}
		names := map[int]string{}
		for _, l := range res.Listeners {
			names[l.PID] = l.Process
		}
		var killed []int
		for _, l := range res.Listeners {
			if seen[l.PID] {
				continue
			}
			seen[l.PID] = true
			switch err := port.Kill(l.PID); {
			case errors.Is(err, port.ErrSystemProcess):
				return output.Fail("PORT_SYSTEM_PROCESS", "Port do Windows (System) giữ, không thể kill", "Thường do IIS/http.sys: tắt dịch vụ tương ứng hoặc đổi port")
			case errors.Is(err, os.ErrPermission):
				return output.NeedAdmin(fmt.Sprintf("Không đủ quyền kill PID %d", l.PID))
			case err != nil:
				return output.Fail("PORT_KILL_FAILED", fmt.Sprintf("Không kill được PID %d: %v", l.PID, err), "Thử kill thủ công: taskkill /PID "+strconv.Itoa(l.PID)+" /F")
			}
			killed = append(killed, l.PID)
		}
		if res, err = port.WaitFree(ctx, n, 2*time.Second); err != nil {
			return netstatErr(err)
		}
		res.Killed = killed
		return p.Success(res, func(w io.Writer) error {
			for _, pid := range killed {
				fmt.Fprintf(w, "Đã kill PID %d (%s).\n", pid, names[pid])
			}
			if res.InUse {
				_, err := fmt.Fprintf(w, "Port %d vẫn còn được dùng.\n", n)
				return err
			}
			_, err := fmt.Fprintf(w, "Port %d giờ đã trống.\n", n)
			return err
		})
	}

	return p.Success(res, func(w io.Writer) error {
		if !res.InUse {
			_, err := fmt.Fprintf(w, "Port %d đang trống.\n", n)
			return err
		}
		fmt.Fprintf(w, "Port %d đang được dùng:\n", n)
		for _, l := range res.Listeners {
			fmt.Fprintf(w, "  PID %d  %s  %s\n", l.PID, l.Process, l.Address)
		}
		return nil
	})
}
