package laralog

import (
	"bytes"
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
)

type Entry struct {
	Time      string   `json:"time"`
	Env       string   `json:"env"`
	Level     string   `json:"level"`
	Message   string   `json:"message"`
	Exception string   `json:"exception,omitempty"`
	File      string   `json:"file,omitempty"`
	Trace     []string `json:"trace,omitempty"`
	Count     int      `json:"count"`
}

// Levels theo thứ tự tăng dần (Monolog).
var Levels = []string{"debug", "info", "notice", "warning", "error", "critical", "alert", "emergency"}

type Filter struct {
	MinLevel string    // rỗng = tất cả
	Since    time.Time // zero = không giới hạn
	Grep     string
	Full     bool   // in đủ stack trace
	Dir      string // thư mục dự án, để rút đường dẫn về tương đối
}

func levelRank(l string) int {
	l = strings.ToLower(l)
	for i, x := range Levels {
		if x == l {
			return i
		}
	}
	return -1
}

var (
	headerRe = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2}[ T]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:[+-]\d{2}:?\d{2}|Z)?)\] (\w+)\.(\w+): (.*)$`)
	excRe    = regexp.MustCompile(`\(([\w\\]+)\(code: \w+\): .*? at (.+?):(\d+)\)`)
	frameRe  = regexp.MustCompile(`^#\d+ `)
)

// ParseTime hiểu timestamp của log (có hoặc không có múi giờ; không có thì coi là giờ local).
func ParseTime(s string) (time.Time, bool) {
	for _, layout := range []string{"2006-01-02 15:04:05.999999999", "2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999Z07:00", "2006-01-02T15:04:05.999999999Z07:00", "2006-01-02T15:04:05Z07:00", "2006-01-02T15:04:05-0700"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func norm(p string) string { return strings.ReplaceAll(strings.ReplaceAll(p, `\\`, `\`), `\`, "/") }

func (f Filter) rel(p string) string {
	p = norm(p)
	if f.Dir != "" {
		return strings.TrimPrefix(p, strings.TrimSuffix(norm(f.Dir), "/")+"/")
	}
	return p
}

// build dựng Entry từ các dòng của một entry (dòng đầu là header).
func build(lines []string, f Filter) (Entry, bool) {
	m := headerRe.FindStringSubmatch(lines[0])
	if m == nil {
		return Entry{}, false
	}
	e := Entry{Time: m[1], Env: m[2], Level: strings.ToUpper(m[3]), Count: 1}
	first := m[4]
	msg := first
	if i := strings.Index(first, ` {"`); i >= 0 { // context JSON ở cuối dòng đầu
		msg = first[:i]
	} else {
		msg = strings.TrimSuffix(first, " []")
	}
	e.Message = strings.TrimSpace(msg)
	text := strings.Join(lines, "\n")
	if x := excRe.FindStringSubmatch(text); x != nil {
		e.Exception, e.File = strings.ReplaceAll(x[1], `\\`, `\`), f.rel(x[2])+":"+x[3]
	}
	var frames []string
	for _, l := range lines[1:] {
		if frameRe.MatchString(l) {
			frames = append(frames, l)
		}
	}
	switch {
	case f.Full:
		// giữ nguyên toàn bộ frame
	case len(frames) > 0:
		var mine []string
		for _, fr := range frames {
			n := norm(fr)
			if strings.Contains(n, "/app/") && !strings.Contains(n, "/vendor/") {
				mine = append(mine, fr)
			}
		}
		if len(mine) > 0 {
			frames = mine[:min(5, len(mine))]
		} else {
			frames = frames[:min(3, len(frames))]
		}
	}
	for _, fr := range frames {
		e.Trace = append(e.Trace, f.relFrame(fr))
	}
	return e, true
}

var frameSplitRe = regexp.MustCompile(`^(#\d+ )([^(]+)(\(.*)$`)

// relFrame rút đường dẫn file trong một frame về tương đối; phần tên class/hàm (có "\") giữ nguyên.
func (f Filter) relFrame(fr string) string {
	m := frameSplitRe.FindStringSubmatch(fr)
	if m == nil {
		return strings.ReplaceAll(fr, `\\`, `\`)
	}
	return m[1] + f.rel(m[2]) + strings.ReplaceAll(m[3], `\\`, `\`)
}

func (f Filter) match(e Entry) bool {
	if f.MinLevel != "" && levelRank(e.Level) < levelRank(f.MinLevel) {
		return false
	}
	if f.Grep != "" && !strings.Contains(strings.ToLower(e.Message), strings.ToLower(f.Grep)) {
		return false
	}
	return true
}

// tooOld: entry cũ hơn --since (log theo thứ tự thời gian nên có thể dừng đọc lùi).
func (f Filter) tooOld(e Entry) bool {
	if f.Since.IsZero() {
		return false
	}
	t, ok := ParseTime(e.Time)
	return ok && t.Before(f.Since)
}

const block = 64 << 10

// readBackward đọc file từ cuối, lùi từng khối 64KB, và gọi fn cho từng entry theo thứ tự mới → cũ.
// fn trả false để dừng. Không bao giờ đọc cả file.
func readBackward(r io.ReaderAt, size int64, f Filter, fn func(Entry) bool) error {
	var carry []byte
	var pending []string // dòng thuộc entry mà header nằm ở khối trước (chưa đọc tới)
	pos := size
	for pos > 0 {
		n := int64(block)
		if pos < n {
			n = pos
		}
		pos -= n
		buf := make([]byte, n)
		if _, err := r.ReadAt(buf, pos); err != nil && err != io.EOF {
			return err
		}
		data := append(buf, carry...)
		carry = nil
		if pos > 0 {
			i := bytes.IndexByte(data, '\n')
			if i < 0 {
				carry = data
				continue
			}
			carry, data = append([]byte(nil), data[:i]...), data[i+1:]
		}
		lines := splitLines(string(data))
		lines = append(lines, pending...)
		end := len(lines)
		for i := len(lines) - 1; i >= 0; i-- {
			if !headerRe.MatchString(lines[i]) {
				continue
			}
			if e, ok := build(lines[i:end], f); ok {
				if f.tooOld(e) {
					return nil
				}
				if f.match(e) && !fn(e) {
					return nil
				}
			}
			end = i
		}
		pending = append([]string(nil), lines[:end]...)
	}
	return nil
}

func splitLines(s string) []string {
	s = strings.TrimRight(s, "\r\n")
	if s == "" {
		return nil
	}
	ls := strings.Split(s, "\n")
	for i := range ls {
		ls[i] = strings.TrimRight(ls[i], "\r")
	}
	return ls
}

// ReadLast trả tối đa n entry mới nhất (theo thứ tự thời gian, cũ → mới) từ danh sách file xếp mới → cũ.
// Entry liên tiếp giống hệt (cùng level + message) được gộp vào Count.
func ReadLast(paths []string, f Filter, n int) ([]Entry, error) {
	var newest []Entry
	for _, p := range paths {
		file, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		st, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, err
		}
		full := false
		err = readBackward(file, st.Size(), f, func(e Entry) bool {
			newest = append(newest, e)
			full = len(newest) >= n
			return !full
		})
		file.Close()
		if err != nil {
			return nil, err
		}
		if full {
			break
		}
	}
	out := make([]Entry, 0, len(newest))
	for i := len(newest) - 1; i >= 0; i-- {
		e := newest[i]
		if k := len(out) - 1; k >= 0 && out[k].Level == e.Level && out[k].Message == e.Message {
			out[k].Count += e.Count
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// Follow theo dõi file như `tail -f` (polling 500ms). resolve trả đường dẫn file hiện tại (kênh daily sang ngày mới thì đổi file).
// File nhỏ lại (xoay vòng/xóa) thì đọc lại từ đầu. Dừng khi ctx hủy.
func Follow(ctx context.Context, resolve func() string, f Filter, emit func(Entry)) error {
	path := resolve()
	var offset int64
	if st, err := os.Stat(path); err == nil {
		offset = st.Size() // chỉ theo dõi phần mới
	}
	var partial string
	var cur []string
	flush := func() {
		if len(cur) > 0 {
			if e, ok := build(cur, f); ok && f.match(e) {
				emit(e)
			}
			cur = nil
		}
	}
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			flush()
			return nil
		case <-tick.C:
		}
		if p := resolve(); p != path {
			flush()
			path, offset, partial = p, 0, ""
		}
		st, err := os.Stat(path)
		if err != nil {
			continue
		}
		if st.Size() < offset {
			flush()
			offset, partial = 0, ""
		}
		if st.Size() == offset {
			continue
		}
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		buf := make([]byte, st.Size()-offset)
		n, _ := file.ReadAt(buf, offset)
		file.Close()
		offset += int64(n)
		data := partial + string(buf[:n])
		idx := strings.LastIndexByte(data, '\n')
		if idx < 0 {
			partial = data
			continue
		}
		partial = data[idx+1:]
		for _, l := range splitLines(data[:idx+1]) {
			if headerRe.MatchString(l) {
				flush()
			}
			if len(cur) > 0 || headerRe.MatchString(l) {
				cur = append(cur, l)
			}
		}
		// entry cuối có thể còn dòng đến sau: để lần poll kế tiếp gộp; nhưng nếu im lặng thì emit ngay
		flush()
	}
}
