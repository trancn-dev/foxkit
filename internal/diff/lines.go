package diff

import (
	"strings"
)

type Op string

const (
	Equal Op = "equal"
	Add   Op = "add"
	Del   Op = "del"
)

type Line struct {
	Op   Op     `json:"op"`
	A    int    `json:"a,omitempty"`
	B    int    `json:"b,omitempty"`
	Text string `json:"text"`
}

type Hunk struct {
	AStart int    `json:"aStart"`
	ALen   int    `json:"aLen"`
	BStart int    `json:"bStart"`
	BLen   int    `json:"bLen"`
	Lines  []Line `json:"lines"`
}

type Options struct {
	Context     int
	IgnoreSpace bool
	IgnoreEOL   bool
	IgnoreCase  bool
	NamesOnly   bool
	Exclude     []string
}

// maxEdit: vượt ngưỡng số thao tác sửa thì dừng Myers (bộ nhớ O(D²)) và coi đoạn giữa là "thay hết".
const maxEdit = 3000

// key chuẩn hóa một dòng theo cờ --ignore-*; chỉ để so sánh, dòng in ra luôn là dòng gốc.
func (o Options) key(s string) string {
	if o.IgnoreEOL {
		s = strings.TrimSuffix(s, "\r")
	}
	if o.IgnoreSpace {
		s = strings.Join(strings.Fields(s), " ")
	}
	if o.IgnoreCase {
		s = strings.ToLower(s)
	}
	return s
}

// myers trả chuỗi thao tác 'e' (giữ), 'd' (xóa), 'a' (thêm) theo thuật toán Myers O(ND); ok=false nếu D vượt maxEdit.
func myers(a, b []string) (ops []byte, ok bool) {
	n, m := len(a), len(b)
	max := n + m
	offset := max + 1
	v := make([]int, 2*max+3)
	var trace [][]int // trace[d] = cửa sổ v (k từ -d-1..d+1) TRƯỚC khi xử lý d
	for d := 0; d <= max && d <= maxEdit; d++ {
		trace = append(trace, append([]int(nil), v[offset-d-1:offset+d+2]...))
		for k := -d; k <= d; k += 2 {
			var x int
			if k == -d || (k != d && v[offset+k-1] < v[offset+k+1]) {
				x = v[offset+k+1]
			} else {
				x = v[offset+k-1] + 1
			}
			y := x - k
			for x < n && y < m && a[x] == b[y] {
				x++
				y++
			}
			v[offset+k] = x
			if x >= n && y >= m {
				return backtrack(trace, n, m), true
			}
		}
	}
	return nil, false
}

func backtrack(trace [][]int, n, m int) []byte {
	var ops []byte
	x, y := n, m
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		at := func(k int) int { return v[k+d+1] }
		k := x - y
		var prevK int
		if k == -d || (k != d && at(k-1) < at(k+1)) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := at(prevK)
		prevY := prevX - prevK
		for x > prevX && y > prevY {
			ops = append(ops, 'e')
			x--
			y--
		}
		if d > 0 {
			if x == prevX {
				ops = append(ops, 'a')
			} else {
				ops = append(ops, 'd')
			}
		}
		x, y = prevX, prevY
	}
	for i, j := 0, len(ops)-1; i < j; i, j = i+1, j-1 {
		ops[i], ops[j] = ops[j], ops[i]
	}
	return ops
}

// Lines so sánh hai dãy dòng và trả dãy thao tác đầy đủ (kể cả dòng giống nhau). Dòng trả về là dòng gốc của a (equal/del) hoặc b (add).
func Lines(a, b []string, opt Options) []Line {
	ka, kb := make([]string, len(a)), make([]string, len(b))
	for i, s := range a {
		ka[i] = opt.key(s)
	}
	for i, s := range b {
		kb[i] = opt.key(s)
	}
	// cắt tiền tố/hậu tố giống nhau trước (file lớn ít thay đổi nhanh hẳn)
	pre := 0
	for pre < len(ka) && pre < len(kb) && ka[pre] == kb[pre] {
		pre++
	}
	suf := 0
	for suf < len(ka)-pre && suf < len(kb)-pre && ka[len(ka)-1-suf] == kb[len(kb)-1-suf] {
		suf++
	}
	midA, midB := ka[pre:len(ka)-suf], kb[pre:len(kb)-suf]
	ops, ok := myers(midA, midB)
	if !ok { // quá khác nhau: coi như thay toàn bộ đoạn giữa
		ops = append(append([]byte(nil), repeat('d', len(midA))...), repeat('a', len(midB))...)
	}
	out := make([]Line, 0, len(a)+len(b))
	ai, bi := 0, 0
	for ; ai < pre; ai, bi = ai+1, bi+1 {
		out = append(out, Line{Op: Equal, A: ai + 1, B: bi + 1, Text: a[ai]})
	}
	for _, op := range ops {
		switch op {
		case 'e':
			out = append(out, Line{Op: Equal, A: ai + 1, B: bi + 1, Text: a[ai]})
			ai, bi = ai+1, bi+1
		case 'd':
			out = append(out, Line{Op: Del, A: ai + 1, Text: a[ai]})
			ai++
		case 'a':
			out = append(out, Line{Op: Add, B: bi + 1, Text: b[bi]})
			bi++
		}
	}
	for ; ai < len(a); ai, bi = ai+1, bi+1 {
		out = append(out, Line{Op: Equal, A: ai + 1, B: bi + 1, Text: a[ai]})
	}
	return out
}

func repeat(c byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return b
}

// ToHunks gom các thay đổi cách nhau ≤ 2×context thành một hunk, kèm context dòng giống nhau mỗi phía.
func ToHunks(ops []Line, context int) []Hunk {
	var changes []int
	for i, l := range ops {
		if l.Op != Equal {
			changes = append(changes, i)
		}
	}
	var hunks []Hunk
	for i := 0; i < len(changes); {
		start, end := changes[i], changes[i]
		j := i + 1
		for j < len(changes) && changes[j]-end <= 2*context+1 {
			end = changes[j]
			j++
		}
		i = j
		from, to := max(0, start-context), min(len(ops)-1, end+context)
		h := Hunk{Lines: ops[from : to+1]}
		for _, l := range h.Lines {
			if l.Op != Add {
				if h.AStart == 0 {
					h.AStart = l.A
				}
				h.ALen++
			}
			if l.Op != Del {
				if h.BStart == 0 {
					h.BStart = l.B
				}
				h.BLen++
			}
		}
		hunks = append(hunks, h)
	}
	return hunks
}
