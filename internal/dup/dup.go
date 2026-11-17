package dup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"foxkit/internal/fsutil"
)

type Options struct {
	Dirs          []string
	MinSize       int64
	Exts          []string
	Workers       int
	IncludeHidden bool
	// Progress (tùy chọn) nhận số file đã quét, gọi từ goroutine quét.
	Progress func(scanned int)
}

type Group struct {
	Size  int64    `json:"size"`
	Hash  string   `json:"hash"`
	Files []string `json:"files"`
}

type Report struct {
	Scanned     int     `json:"scanned"`
	Groups      []Group `json:"groups"`
	WastedBytes int64   `json:"wastedBytes"`
	Skipped     int     `json:"skipped"`
	DurationMs  int64   `json:"durationMs"`
}

const partialBytes = 4096

type hashResult struct {
	path string
	hash string
	err  error
}

// Find: gom theo size → băm 4KB đầu → băm toàn file; chỉ file qua cả ba tầng mới là trùng.
func Find(ctx context.Context, opt Options) (Report, error) {
	start := time.Now()
	if opt.Workers < 1 {
		opt.Workers = runtime.NumCPU()
	}
	bySize, scanned, skipped, err := scan(ctx, opt)
	if err != nil {
		return Report{}, err
	}
	rep := Report{Scanned: scanned, Skipped: skipped, Groups: []Group{}}

	// tầng 2: gom theo size
	var cands []string
	sizeOf := map[string]int64{}
	for size, paths := range bySize {
		if len(paths) < 2 {
			continue
		}
		for _, p := range paths {
			cands = append(cands, p)
			sizeOf[p] = size
		}
	}
	// tầng 3: băm 4KB đầu
	partial, sk, err := hashAll(ctx, cands, opt.Workers, partialBytes)
	if err != nil {
		return Report{}, err
	}
	rep.Skipped += sk
	type key struct {
		size int64
		hash string
	}
	byPartial := map[key][]string{}
	for p, h := range partial {
		k := key{sizeOf[p], h}
		byPartial[k] = append(byPartial[k], p)
	}
	var full []string
	for _, ps := range byPartial {
		if len(ps) >= 2 {
			full = append(full, ps...)
		}
	}
	// tầng 4: băm toàn file
	hashes, sk, err := hashAll(ctx, full, opt.Workers, -1)
	if err != nil {
		return Report{}, err
	}
	rep.Skipped += sk
	byFull := map[key][]string{}
	for p, h := range hashes {
		k := key{sizeOf[p], h}
		byFull[k] = append(byFull[k], p)
	}
	for k, ps := range byFull {
		if len(ps) < 2 {
			continue
		}
		sort.Strings(ps)
		rep.Groups = append(rep.Groups, Group{Size: k.size, Hash: k.hash, Files: ps})
		rep.WastedBytes += k.size * int64(len(ps)-1)
	}
	sort.Slice(rep.Groups, func(i, j int) bool {
		wi := rep.Groups[i].Size * int64(len(rep.Groups[i].Files)-1)
		wj := rep.Groups[j].Size * int64(len(rep.Groups[j].Files)-1)
		if wi != wj {
			return wi > wj
		}
		return rep.Groups[i].Files[0] < rep.Groups[j].Files[0]
	})
	rep.DurationMs = time.Since(start).Milliseconds()
	return rep, nil
}

// scan duyệt các thư mục và gom file theo size. Một file xuất hiện qua hai dir lồng nhau chỉ tính một lần.
func scan(ctx context.Context, opt Options) (map[int64][]string, int, int, error) {
	exts := map[string]bool{}
	for _, e := range opt.Exts {
		exts[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e), "."))] = true
	}
	bySize := map[int64][]string{}
	seen := map[string]bool{}
	scanned, skipped := 0, 0
	for _, root := range opt.Dirs {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if err != nil {
				skipped++
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				if p != root && !opt.IncludeHidden && (fsutil.IsSkipDir(d.Name(), fsutil.DefaultSkipDirs) || fsutil.IsHidden(p)) {
					return fs.SkipDir
				}
				return nil
			}
			if fsutil.Skippable(d) || !d.Type().IsRegular() {
				return nil
			}
			if !opt.IncludeHidden && fsutil.IsHidden(p) {
				return nil
			}
			if len(exts) > 0 && !exts[strings.ToLower(strings.TrimPrefix(filepath.Ext(p), "."))] {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				skipped++
				return nil
			}
			if info.Size() < opt.MinSize {
				return nil
			}
			abs, err := filepath.Abs(p)
			if err != nil {
				skipped++
				return nil
			}
			key := strings.ToLower(filepath.Clean(abs))
			if seen[key] {
				return nil
			}
			seen[key] = true
			bySize[info.Size()] = append(bySize[info.Size()], abs)
			scanned++
			if opt.Progress != nil && scanned%200 == 0 {
				opt.Progress(scanned)
			}
			return nil
		})
		if err != nil {
			return nil, 0, 0, err
		}
	}
	return bySize, scanned, skipped, nil
}

// hashAll băm song song bằng worker pool; limit<0 = toàn file, ngược lại băm limit byte đầu.
// Chỉ goroutine gọi hàm này ghi vào map nên không cần mutex.
func hashAll(ctx context.Context, paths []string, workers int, limit int64) (map[string]string, int, error) {
	jobs := make(chan string)
	results := make(chan hashResult)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range jobs {
				h, err := hashFile(p, limit)
				select {
				case results <- hashResult{p, h, err}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	var canceled atomic.Bool
	go func() {
		defer close(jobs)
		for _, p := range paths {
			select {
			case jobs <- p:
			case <-ctx.Done():
				canceled.Store(true)
				return
			}
		}
	}()
	go func() { wg.Wait(); close(results) }()
	out := make(map[string]string, len(paths))
	skipped := 0
	for r := range results {
		if r.err != nil {
			skipped++
			continue
		}
		out[r.path] = r.hash
	}
	if ctx.Err() != nil {
		return nil, 0, ctx.Err()
	}
	return out, skipped, nil
}

func hashFile(path string, limit int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	var r io.Reader = f
	if limit >= 0 {
		r = io.LimitReader(f, limit)
	}
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
