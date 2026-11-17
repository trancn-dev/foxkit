package winexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Run chạy lệnh có timeout, trả stdout. Lỗi gói kèm stderr.
func Run(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	hideWindow(cmd)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return out.Bytes(), fmt.Errorf("%s: quá thời gian %s: %w", name, timeout, ctx.Err())
		}
		return out.Bytes(), fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

// RunCombined như Run nhưng trả stdout+stderr gộp và bỏ qua exit code khác 0 (dùng cho `--version`
// vì nhiều tool in phiên bản ra stderr hoặc thoát mã lạ). Chỉ trả lỗi khi không khởi chạy được/quá giờ.
func RunCombined(ctx context.Context, timeout time.Duration, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	hideWindow(cmd)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, nil
	}
	return out, err
}
