package zipx

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"foxkit/internal/rename"
)

type UnpackResult struct {
	To    string `json:"to"`
	Files int    `json:"files"`
	Bytes int64  `json:"bytes"`
}

const maxTotal = int64(20) << 30 // 20GB

// DefaultTo: `<tên zip không đuôi>` cạnh file zip.
func DefaultTo(zipPath string) string {
	return strings.TrimSuffix(zipPath, filepath.Ext(zipPath))
}

// safeDest kiểm entry không thoát khỏi thư mục đích (Zip Slip): không tuyệt đối, không ".." thoát ra, không ký tự/tên dành riêng của Windows.
func safeDest(to, name string) (string, error) {
	if name == "" || strings.HasPrefix(name, "/") || strings.HasPrefix(name, `\`) || filepath.IsAbs(name) {
		return "", fmt.Errorf("%w: %q", ErrUnsafePath, name)
	}
	n := strings.ReplaceAll(name, `\`, "/") // zip lẫn dấu \ cũng bị coi như dấu phân cách
	for _, seg := range strings.Split(strings.TrimSuffix(n, "/"), "/") {
		if seg == ".." {
			return "", fmt.Errorf("%w: %q", ErrUnsafePath, name)
		}
		if seg == "" || seg == "." {
			continue
		}
		if !rename.ValidName(seg) {
			return "", fmt.Errorf("%w: %q (tên không hợp lệ trên Windows)", ErrUnsafePath, name)
		}
	}
	dest := filepath.Join(to, filepath.FromSlash(n))
	rel, err := filepath.Rel(to, dest)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: %q", ErrUnsafePath, name)
	}
	return dest, nil
}

// Unpack giải nén an toàn: kiểm mọi entry TRƯỚC khi ghi file nào (đường dẫn, symlink, zip bomb, file đã tồn tại).
func Unpack(ctx context.Context, zipPath, to string, force bool) (UnpackResult, error) {
	res := UnpackResult{To: to}
	st, err := os.Stat(zipPath)
	if err != nil {
		return res, fmt.Errorf("%w: %s", ErrNotFound, zipPath)
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return res, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	defer r.Close()
	absTo, err := filepath.Abs(to)
	if err != nil {
		return res, err
	}

	limit := min(st.Size()*20, maxTotal)
	var total int64
	dests := make([]string, len(r.File))
	var exists []string
	for i, f := range r.File {
		dest, err := safeDest(absTo, f.Name)
		if err != nil {
			return res, err
		}
		if f.Mode()&fs.ModeSymlink != 0 {
			return res, fmt.Errorf("%w: %q là symlink", ErrUnsafePath, f.Name)
		}
		dests[i] = dest
		total += int64(f.UncompressedSize64)
		if total > limit {
			return res, fmt.Errorf("%w: %d byte > giới hạn %d", ErrTooLarge, total, limit)
		}
		if !f.FileInfo().IsDir() && !force {
			if _, err := os.Stat(dest); err == nil {
				exists = append(exists, f.Name)
			}
		}
	}
	if len(exists) > 0 {
		return res, fmt.Errorf("%w: %s", ErrExists, strings.Join(exists, ", "))
	}

	var written int64
	for i, f := range r.File {
		if err := ctx.Err(); err != nil {
			return res, err
		}
		dest := dests[i]
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return res, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return res, err
		}
		src, err := f.Open()
		if err != nil {
			return res, fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			src.Close()
			return res, err
		}
		// LimitReader: header có thể nói dối kích thước, giới hạn theo số byte thực sự đọc ra
		n, cerr := io.Copy(out, io.LimitReader(src, limit-written+1))
		src.Close()
		if err := out.Close(); err != nil && cerr == nil {
			cerr = err
		}
		written += n
		if written > limit {
			return res, fmt.Errorf("%w: vượt giới hạn %d byte khi giải nén", ErrTooLarge, limit)
		}
		if cerr != nil {
			return res, fmt.Errorf("%w: %v", ErrInvalid, cerr)
		}
		_ = os.Chtimes(dest, f.Modified, f.Modified) // đặt lại ngày sửa; thất bại không ảnh hưởng nội dung
		res.Files++
		res.Bytes += n
	}
	res.To = absTo
	return res, nil
}
