package lara

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"foxkit/internal/env"
	"foxkit/internal/winexec"
)

// IsProject: có artisan và composer.json chứa laravel/framework.
func IsProject(dir string) bool {
	if st, err := os.Stat(filepath.Join(dir, "artisan")); err != nil || st.IsDir() {
		return false
	}
	b, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	return err == nil && bytes.Contains(b, []byte("laravel/framework"))
}

// LogFiles trả file log mới → cũ: laravel.log (single) rồi laravel-YYYY-MM-DD.log (daily) giảm dần theo tên.
func LogFiles(dir string) []string {
	logs := filepath.Join(dir, "storage", "logs")
	var out []string
	if p := filepath.Join(logs, "laravel.log"); fileExists(p) {
		out = append(out, p)
	}
	daily, _ := filepath.Glob(filepath.Join(logs, "laravel-*.log")) // Glob chỉ lỗi khi pattern sai; pattern cố định
	sort.Sort(sort.Reverse(sort.StringSlice(daily)))
	return append(out, daily...)
}

func fileExists(p string) bool { st, err := os.Stat(p); return err == nil && !st.IsDir() }

// NewKey: 32 byte ngẫu nhiên dạng "base64:..." (như `php artisan key:generate` với AES-256-CBC).
func NewKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "base64:" + base64.StdEncoding.EncodeToString(b), nil
}

var ErrKeyExists = errors.New("APP_KEY đã có giá trị")

// ApplyKey ghi APP_KEY vào .env, giữ nguyên format; backup `.env.foxkit.bak`, ghi tmp rồi rename.
// APP_KEY đã có giá trị mà không force → ErrKeyExists.
func ApplyKey(dir, key string, force bool) (file string, err error) {
	file = filepath.Join(dir, ".env")
	f, err := env.ParseFile(file)
	if err != nil {
		return file, err
	}
	if v, _ := f.Get("APP_KEY"); v != "" && !force {
		return file, ErrKeyExists
	}
	f.Set("APP_KEY", key)
	old, err := os.ReadFile(file)
	if err != nil {
		return file, err
	}
	if err := os.WriteFile(file+".foxkit.bak", old, 0o600); err != nil {
		return file, err
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return file, err
	}
	tmp := file + ".foxkit.tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return file, err
	}
	if err := os.Rename(tmp, file); err != nil {
		_ = os.Remove(tmp) // dọn file tạm khi rename lỗi
		return file, err
	}
	return file, nil
}

// artisan chạy `php artisan <args>` trong dir; tách ra để test fallback mà không cần PHP.
var artisan = func(ctx context.Context, dir string, args ...string) ([]byte, int, error) {
	php, err := exec.LookPath("php")
	if err != nil {
		return nil, 0, ErrPHPNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, php, append([]string{"artisan"}, args...)...)
	cmd.Dir = dir
	winexec.Hide(cmd)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return out, exit.ExitCode(), nil
	}
	return out, 0, err
}

var ErrPHPNotFound = errors.New("không tìm thấy php")

type ClearResult struct {
	Ran      []string `json:"ran"`
	Output   string   `json:"output"`
	ExitCode int      `json:"exitCode"`
}

// Clear chạy `optimize:clear`; Laravel cũ không có lệnh này thì chạy lần lượt config/cache/route/view:clear.
func Clear(ctx context.Context, dir string) (ClearResult, error) {
	res := ClearResult{Ran: []string{}}
	run := func(cmd string) (int, error) {
		out, code, err := artisan(ctx, dir, cmd)
		res.Ran = append(res.Ran, "php artisan "+cmd)
		res.Output += string(out)
		return code, err
	}
	code, err := run("optimize:clear")
	if err != nil {
		return res, err
	}
	if code != 0 && (strings.Contains(res.Output, "not defined") || strings.Contains(res.Output, "not found")) {
		res.Output = ""
		for _, c := range []string{"config:clear", "cache:clear", "route:clear", "view:clear"} {
			if code, err = run(c); err != nil || code != 0 {
				break
			}
		}
	}
	res.ExitCode = code
	return res, err
}
