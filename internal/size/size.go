package size

import (
	"container/heap"
	"context"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"

	"foxkit/internal/fsutil"
)

type Item struct {
	Name  string  `json:"name"`
	Path  string  `json:"path"`
	Bytes int64   `json:"bytes"`
	Files int     `json:"files"`
	IsDir bool    `json:"isDir"`
	Pct   float64 `json:"pct"`
}

type Report struct {
	Path   string `json:"path"`
	Total  int64  `json:"total"`
	Items  []Item `json:"items"`
	Denied int    `json:"denied"`
}

func pct(part, total int64) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(part)/float64(total)*1000) / 10
}

// ByChildren: dung lượng từng con trực tiếp của dir, giảm dần, cắt top, phần còn lại gộp một dòng.
func ByChildren(ctx context.Context, dir string, top int) (Report, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Report{}, err
	}
	items := make([]Item, len(entries))
	var denied int
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.NumCPU()*2) // semaphore: tối đa N thư mục quét đồng thời
	for i, e := range entries {
		p := filepath.Join(dir, e.Name())
		items[i] = Item{Name: e.Name(), Path: p, IsDir: e.IsDir()}
		switch {
		case fsutil.Skippable(e):
			continue
		case e.IsDir():
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, p string) {
				defer wg.Done()
				defer func() { <-sem }()
				st, _ := fsutil.DirSize(ctx, p) // lỗi duy nhất là ctx bị hủy, kiểm ở dưới
				items[i].Bytes, items[i].Files = st.Bytes, st.Files
				mu.Lock()
				denied += st.Denied
				mu.Unlock()
			}(i, p)
		default:
			if info, err := e.Info(); err == nil {
				items[i].Bytes, items[i].Files = info.Size(), 1
			} else {
				denied++
			}
		}
	}
	wg.Wait()
	if ctx.Err() != nil {
		return Report{}, ctx.Err()
	}
	var total int64
	for _, it := range items {
		total += it.Bytes
	}
	sort.SliceStable(items, func(a, b int) bool { return items[a].Bytes > items[b].Bytes })
	out := make([]Item, 0, top+1)
	if len(items) > top {
		var rest int64
		var files int
		for _, it := range items[top:] {
			rest += it.Bytes
			files += it.Files
		}
		out = append(out, items[:top]...)
		out = append(out, Item{Name: fmt.Sprintf("(còn lại %d mục)", len(items)-top), Bytes: rest, Files: files})
	} else {
		out = append(out, items...)
	}
	for i := range out {
		out[i].Pct = pct(out[i].Bytes, total)
	}
	return Report{Path: dir, Total: total, Items: out, Denied: denied}, nil
}

// minHeap giữ top N file lớn nhất: phần tử gốc là file NHỎ nhất trong N file đang giữ.
type minHeap []Item

func (h minHeap) Len() int           { return len(h) }
func (h minHeap) Less(i, j int) bool { return h[i].Bytes < h[j].Bytes }
func (h minHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *minHeap) Push(x any)        { *h = append(*h, x.(Item)) }
func (h *minHeap) Pop() any {
	old := *h
	n := len(old)
	it := old[n-1]
	*h = old[:n-1]
	return it
}

// LargestFiles: top N file lớn nhất trong cả cây, bộ nhớ O(N).
func LargestFiles(ctx context.Context, dir string, top int) (Report, error) {
	h := &minHeap{}
	var total int64
	var denied int
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			denied++
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if fsutil.Skippable(d) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			denied++
			return nil
		}
		total += info.Size()
		it := Item{Name: d.Name(), Path: p, Bytes: info.Size(), Files: 1}
		if h.Len() < top {
			heap.Push(h, it)
		} else if it.Bytes > (*h)[0].Bytes {
			(*h)[0] = it
			heap.Fix(h, 0)
		}
		return nil
	})
	if err != nil {
		return Report{}, err
	}
	items := make([]Item, h.Len())
	copy(items, *h)
	sort.Slice(items, func(a, b int) bool { return items[a].Bytes > items[b].Bytes })
	for i := range items {
		items[i].Pct = pct(items[i].Bytes, total)
	}
	return Report{Path: dir, Total: total, Items: items, Denied: denied}, nil
}
