package temp

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"foxkit/internal/fsutil"
)

type Report struct {
	Dir            string `json:"dir"`
	Applied        bool   `json:"applied"`
	TotalBytes     int64  `json:"totalBytes"`
	Candidates     int    `json:"candidates"`
	CandidateBytes int64  `json:"candidateBytes"`
	Deleted        int    `json:"deleted"`
	FreedBytes     int64  `json:"freedBytes"`
	Locked         int    `json:"locked"`
}

type candidate struct {
	path  string
	bytes int64
}

// newest: thời gian sửa mới nhất của mọi file trong thư mục (không có file nào thì dùng chính thư mục).
func newest(dir string, own time.Time) time.Time {
	latest := time.Time{}
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error { // lỗi từng mục bỏ qua, callback luôn trả nil
		if err != nil || !d.Type().IsRegular() {
			return nil
		}
		if info, err := d.Info(); err == nil && info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	if latest.IsZero() {
		return own
	}
	return latest
}

// Run xem/dọn thư mục tạm. Mục đang bị dùng (xóa lỗi) chỉ đếm vào Locked, không phải lỗi của lệnh.
func Run(ctx context.Context, dir string, olderThan time.Duration, apply bool, now time.Time) (Report, error) {
	rep := Report{Dir: dir, Applied: apply}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return rep, err
	}
	cutoff := now.Add(-olderThan)
	var cands []candidate
	for _, e := range entries {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		p := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil || fsutil.Skippable(e) {
			continue
		}
		var size int64
		mod := info.ModTime()
		if e.IsDir() {
			st, derr := fsutil.DirSize(ctx, p)
			if derr != nil {
				return rep, derr
			}
			size, mod = st.Bytes, newest(p, mod)
		} else {
			size = info.Size()
		}
		rep.TotalBytes += size
		if mod.Before(cutoff) {
			cands = append(cands, candidate{p, size})
			rep.Candidates++
			rep.CandidateBytes += size
		}
	}
	if !apply {
		return rep, nil
	}
	for _, c := range cands {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		if err := os.RemoveAll(c.path); err == nil {
			rep.Deleted++
			rep.FreedBytes += c.bytes
			continue
		}
		rep.Locked++
		if st, err := fsutil.DirSize(ctx, c.path); err == nil && st.Bytes < c.bytes { // xóa được một phần
			rep.FreedBytes += c.bytes - st.Bytes
		}
	}
	return rep, nil
}
