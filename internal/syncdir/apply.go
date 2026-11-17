package syncdir

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Event string `json:"event"` // progress | done
	Done  int    `json:"done"`
	Total int    `json:"total"`
	Bytes int64  `json:"bytes"`
	Path  string `json:"path,omitempty"`
}

// ctxReader dừng sao chép ngay khi ctx bị hủy (Ctrl+C), kể cả giữa lúc chép file lớn.
type ctxReader struct {
	ctx context.Context
	r   io.Reader
}

func (c ctxReader) Read(p []byte) (int, error) {
	if err := c.ctx.Err(); err != nil {
		return 0, err
	}
	return c.r.Read(p)
}

// copyAtomic chép src → dst qua `<dst>.foxkit-tmp` rồi rename: file đích không bao giờ ở trạng thái dở dang. Giữ mtime nguồn.
func copyAtomic(ctx context.Context, src, dst string) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return 0, err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return 0, err
	}
	tmp := dst + tmpExt
	out, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(out, ctxReader{ctx, in})
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chtimes(tmp, info.ModTime(), info.ModTime())
	}
	if err == nil {
		err = os.Rename(tmp, dst) // Windows: MoveFileEx có REPLACE_EXISTING nên đè được file cũ
	}
	if err != nil {
		_ = os.Remove(tmp) // không để lại .foxkit-tmp
		return 0, err
	}
	return n, nil
}

// Apply thực thi kế hoạch: mkdir tuần tự, copy/update bằng 4 luồng, delete = chuyển vào `.foxkit-trash\<timestamp>\...`.
// Lỗi từng file vào p.Failed (tiếp tục); trả lỗi ctx nếu bị hủy.
func Apply(ctx context.Context, p *Plan, opt Options, emit func(Event)) error {
	now := time.Now
	if opt.Now != nil {
		now = opt.Now
	}
	trash := filepath.Join(p.Dst, trashDir, now().Format("20060102-150405"))
	total := 0
	for _, a := range p.Actions {
		if a.Op != "mkdir" {
			total++
		}
	}
	var mu sync.Mutex
	done := 0
	var doneBytes int64
	fail := func(path string, err error) {
		mu.Lock()
		p.Failed = append(p.Failed, Failure{Path: path, Reason: err.Error()})
		mu.Unlock()
	}
	progress := func(path string, n int64) {
		mu.Lock()
		done++
		doneBytes += n
		d, b := done, doneBytes
		mu.Unlock()
		if emit != nil {
			emit(Event{Event: "progress", Done: d, Total: total, Bytes: b, Path: path})
		}
	}

	for _, a := range p.Actions {
		if a.Op == "mkdir" {
			if err := os.MkdirAll(filepath.Join(p.Dst, filepath.FromSlash(a.Path)), 0o755); err != nil {
				return fmt.Errorf("không tạo được thư mục %q: %w", a.Path, err)
			}
		}
	}
	jobs := make(chan Action)
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range jobs {
				s, d := filepath.Join(p.Src, filepath.FromSlash(a.Path)), filepath.Join(p.Dst, filepath.FromSlash(a.Path))
				switch a.Op {
				case "copy", "update":
					n, err := copyAtomic(ctx, s, d)
					if err != nil {
						if ctx.Err() == nil {
							fail(a.Path, err)
						}
						continue
					}
					progress(a.Path, n)
				case "delete":
					target := filepath.Join(trash, filepath.FromSlash(a.Path))
					if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
						fail(a.Path, err)
						continue
					}
					if err := os.Rename(d, target); err != nil {
						fail(a.Path, err)
						continue
					}
					progress(a.Path, 0)
				}
			}
		}()
	}
loop:
	for _, a := range p.Actions {
		if a.Op == "mkdir" {
			continue
		}
		select {
		case jobs <- a:
		case <-ctx.Done():
			break loop
		}
	}
	close(jobs)
	wg.Wait()
	p.Applied = true
	if emit != nil {
		mu.Lock()
		emit(Event{Event: "done", Done: done, Total: total, Bytes: doneBytes})
		mu.Unlock()
	}
	return ctx.Err()
}
