package down

import (
	"fmt"
	"strings"
	"time"
)

// speedometer tính tốc độ trung bình trượt trong cửa sổ 3 giây.
type speedometer struct {
	samples []sample
}

type sample struct {
	t     time.Time
	bytes int64
}

func (s *speedometer) add(t time.Time, bytes int64) {
	s.samples = append(s.samples, sample{t, bytes})
	cut := 0
	for cut < len(s.samples)-1 && t.Sub(s.samples[cut].t) > 3*time.Second {
		cut++
	}
	s.samples = s.samples[cut:]
}

func (s *speedometer) rate() int64 {
	if len(s.samples) < 2 {
		return 0
	}
	first, last := s.samples[0], s.samples[len(s.samples)-1]
	dt := last.t.Sub(first.t).Seconds()
	if dt <= 0 {
		return 0
	}
	return int64(float64(last.bytes-first.bytes) / dt)
}

// Bar vẽ thanh tiến độ: [███████░░░░░] 58%  (width ô). total <= 0 thì trả rỗng (không biết tổng).
func Bar(done, total int64, width int) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	pct := float64(done) / float64(total)
	if pct > 1 {
		pct = 1
	}
	n := int(pct*float64(width) + 0.5)
	return fmt.Sprintf("[%s%s] %3.0f%%", strings.Repeat("█", n), strings.Repeat("░", width-n), pct*100)
}

// ETA: thời gian còn lại dạng "1m02s"; không tính được thì trả rỗng.
func ETA(done, total, bytesPerSec int64) string {
	if total <= 0 || bytesPerSec <= 0 || done >= total {
		return ""
	}
	d := time.Duration((total-done)/bytesPerSec) * time.Second
	if d >= time.Hour {
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}
