package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"foxkit/internal/winexec"
)

var ErrNotFound = errors.New("không tìm thấy git")

// Available: git có trong PATH không.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Run chạy `git -C <dir> ...`. GIT_OPTIONAL_LOCKS=0 để lệnh chỉ-đọc không tạo lock (tránh xung đột với IDE đang mở repo).
// Lỗi gồm cả stderr.
func Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	winexec.Hide(cmd)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, ErrNotFound
		}
		return out.Bytes(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errb.String()))
	}
	return out.Bytes(), nil
}

type Status struct {
	Branch    string `json:"branch"`
	Upstream  string `json:"upstream"`
	Ahead     int    `json:"ahead"`
	Behind    int    `json:"behind"`
	Staged    int    `json:"staged"`
	Modified  int    `json:"modified"`
	Untracked int    `json:"untracked"`
	Conflicts int    `json:"conflicts"`
	Stashes   int    `json:"stashes"`
}

// ParseStatusV2 đọc output của `git status --porcelain=v2 --branch` (định dạng dành cho máy, ổn định qua các phiên bản).
func ParseStatusV2(out []byte) (Status, error) {
	var s Status
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			s.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.upstream "):
			s.Upstream = strings.TrimPrefix(line, "# branch.upstream ")
		case strings.HasPrefix(line, "# branch.ab "):
			f := strings.Fields(strings.TrimPrefix(line, "# branch.ab "))
			if len(f) != 2 {
				return s, fmt.Errorf("dòng branch.ab sai: %q", line)
			}
			a, err1 := strconv.Atoi(strings.TrimPrefix(f[0], "+"))
			b, err2 := strconv.Atoi(strings.TrimPrefix(f[1], "-"))
			if err1 != nil || err2 != nil {
				return s, fmt.Errorf("dòng branch.ab sai: %q", line)
			}
			s.Ahead, s.Behind = a, b
		case strings.HasPrefix(line, "1 ") || strings.HasPrefix(line, "2 "):
			if len(line) >= 4 {
				if line[2] != '.' {
					s.Staged++
				}
				if line[3] != '.' {
					s.Modified++
				}
			}
		case strings.HasPrefix(line, "u "):
			s.Conflicts++
		case strings.HasPrefix(line, "? "):
			s.Untracked++
		}
	}
	return s, nil
}
