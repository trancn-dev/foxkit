package csvx

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Table struct {
	Header   []string
	Rows     [][]string
	Delim    rune
	HadBOM   bool
	Warnings []string
}

var ErrEncoding = errors.New("file không phải UTF-8")

// ColError: cột không tồn tại; Have là danh sách cột hiện có.
type ColError struct {
	Name string
	Have []string
}

func (e *ColError) Error() string {
	return fmt.Sprintf("không có cột %q (các cột có: %s)", e.Name, strings.Join(e.Have, ", "))
}

// WhereError: điều kiện --where sai cú pháp.
type WhereError struct{ Expr string }

func (e *WhereError) Error() string {
	return fmt.Sprintf("điều kiện không hợp lệ: %q", e.Expr)
}

// DetectDelim đoán dấu phân cách từ vài dòng đầu: đếm `, ; \t |` ngoài dấu nháy, chọn dấu xuất hiện đều nhất.
func DetectDelim(sample []byte) rune {
	cands := []rune{',', ';', '\t', '|'}
	lines := strings.Split(strings.ReplaceAll(string(sample), "\r\n", "\n"), "\n")
	var use []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			use = append(use, l)
		}
		if len(use) == 5 {
			break
		}
	}
	best, bestScore := ',', -1
	for _, c := range cands {
		counts := make([]int, len(use))
		total := 0
		for i, l := range use {
			inQuote := false
			for _, r := range l {
				switch {
				case r == '"':
					inQuote = !inQuote
				case r == c && !inQuote:
					counts[i]++
				}
			}
			total += counts[i]
		}
		if total == 0 {
			continue
		}
		consistent := true
		for _, n := range counts {
			if n != counts[0] {
				consistent = false
			}
		}
		score := total
		if consistent {
			score += 1 << 20 // xuất hiện đều ở mọi dòng thắng dấu chỉ xuất hiện nhiều
		}
		if score > bestScore {
			best, bestScore = c, score
		}
	}
	return best
}

// Read đọc CSV. delim = 0 → tự đoán. Bỏ BOM; không phải UTF-8 hợp lệ → ErrEncoding.
// Dòng thiếu/thừa cột được đệm cho đủ và ghi vào Warnings.
func Read(r io.Reader, delim rune, noHeader bool) (*Table, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	t := &Table{}
	if bytes.HasPrefix(raw, []byte("\xef\xbb\xbf")) {
		t.HadBOM = true
		raw = raw[3:]
	}
	if !utf8.Valid(raw) {
		return nil, ErrEncoding
	}
	if delim == 0 {
		sample := raw
		if len(sample) > 8192 {
			sample = sample[:8192]
		}
		delim = DetectDelim(sample)
	}
	t.Delim = delim
	cr := csv.NewReader(bytes.NewReader(raw))
	cr.Comma = delim
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1
	records, err := cr.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		t.Header, t.Rows = []string{}, [][]string{}
		return t, nil
	}
	width := 0
	for _, rec := range records {
		width = max(width, len(rec))
	}
	if noHeader {
		t.Header = make([]string, width)
		for i := range t.Header {
			t.Header[i] = "c" + strconv.Itoa(i+1)
		}
	} else {
		t.Header = append([]string{}, records[0]...)
		for len(t.Header) < width {
			t.Header = append(t.Header, "c"+strconv.Itoa(len(t.Header)+1))
		}
		records = records[1:]
	}
	uneven := 0
	t.Rows = make([][]string, 0, len(records))
	for _, rec := range records {
		if len(rec) != width {
			uneven++
			rec = append(append([]string{}, rec...), make([]string, width-len(rec))...)
		}
		t.Rows = append(t.Rows, rec)
	}
	if uneven > 0 {
		t.Warnings = append(t.Warnings, fmt.Sprintf("%d dòng thiếu cột, đã đệm ô rỗng", uneven))
	}
	return t, nil
}

func (t *Table) colIndex(name string) (int, bool) {
	if n, err := strconv.Atoi(name); err == nil && n >= 1 && n <= len(t.Header) {
		return n - 1, true
	}
	for i, h := range t.Header {
		if strings.EqualFold(h, name) {
			return i, true
		}
	}
	return 0, false
}

// Select chọn cột theo tên hoặc số thứ tự 1-based, giữ thứ tự nhập.
func (t *Table) Select(cols []string) (*Table, error) {
	idx := make([]int, len(cols))
	for i, c := range cols {
		j, ok := t.colIndex(strings.TrimSpace(c))
		if !ok {
			return nil, &ColError{Name: c, Have: t.Header}
		}
		idx[i] = j
	}
	out := &Table{Delim: t.Delim, HadBOM: t.HadBOM, Warnings: t.Warnings, Header: make([]string, len(idx)), Rows: make([][]string, len(t.Rows))}
	for i, j := range idx {
		out.Header[i] = t.Header[j]
	}
	for r, row := range t.Rows {
		out.Rows[r] = make([]string, len(idx))
		for i, j := range idx {
			out.Rows[r][i] = row[j]
		}
	}
	return out, nil
}

type cond struct {
	col, op, val string
}

var ops = []string{"!=", ">=", "<=", "=", "~", ">", "<"}

func parseWhere(expr string) ([]cond, error) {
	var out []cond
	for _, part := range strings.Split(expr, ",") {
		part = strings.TrimSpace(part)
		found := false
		for _, op := range ops {
			if i := strings.Index(part, op); i > 0 {
				out = append(out, cond{strings.TrimSpace(part[:i]), op, strings.TrimSpace(part[i+len(op):])})
				found = true
				break
			}
		}
		if !found {
			return nil, &WhereError{Expr: part}
		}
	}
	return out, nil
}

func (c cond) match(cell string) bool {
	a, errA := strconv.ParseFloat(strings.TrimSpace(cell), 64)
	b, errB := strconv.ParseFloat(c.val, 64)
	numeric := errA == nil && errB == nil
	switch c.op {
	case "~":
		return strings.Contains(strings.ToLower(cell), strings.ToLower(c.val))
	case "=":
		if numeric {
			return a == b
		}
		return strings.EqualFold(cell, c.val)
	case "!=":
		if numeric {
			return a != b
		}
		return !strings.EqualFold(cell, c.val)
	}
	if numeric {
		switch c.op {
		case ">":
			return a > b
		case "<":
			return a < b
		case ">=":
			return a >= b
		case "<=":
			return a <= b
		}
	}
	l, v := strings.ToLower(cell), strings.ToLower(c.val)
	switch c.op {
	case ">":
		return l > v
	case "<":
		return l < v
	case ">=":
		return l >= v
	default:
		return l <= v
	}
}

// Where lọc dòng: `col=v`, `col!=v`, `col~sub`, `col>10`, `col<10`...; nhiều điều kiện cách bằng "," là AND.
func (t *Table) Where(expr string) (*Table, error) {
	conds, err := parseWhere(expr)
	if err != nil {
		return nil, err
	}
	idx := make([]int, len(conds))
	for i, c := range conds {
		j, ok := t.colIndex(c.col)
		if !ok {
			return nil, &ColError{Name: c.col, Have: t.Header}
		}
		idx[i] = j
	}
	out := &Table{Header: t.Header, Delim: t.Delim, HadBOM: t.HadBOM, Warnings: t.Warnings, Rows: [][]string{}}
	for _, row := range t.Rows {
		ok := true
		for i, c := range conds {
			if !c.match(row[idx[i]]) {
				ok = false
				break
			}
		}
		if ok {
			out.Rows = append(out.Rows, row)
		}
	}
	return out, nil
}

type ColStat struct {
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Empty  int    `json:"empty"`
	Unique string `json:"unique"` // ">100000" khi vượt giới hạn đếm
	Kind   string `json:"kind"`
	Min    string `json:"min,omitempty"`
	Max    string `json:"max,omitempty"`
}

const uniqueCap = 100000

// Stats: mỗi cột đếm dòng/rỗng/duy nhất, đoán kiểu theo ≥95% giá trị không rỗng, min/max cho số và ngày.
func (t *Table) Stats() []ColStat {
	out := make([]ColStat, len(t.Header))
	for c, name := range t.Header {
		s := ColStat{Name: name, Count: len(t.Rows)}
		uniq := map[string]struct{}{}
		over := false
		ints, floats, dates, nonEmpty := 0, 0, 0, 0
		var minF, maxF float64
		var minD, maxD string
		first := true
		for _, row := range t.Rows {
			v := strings.TrimSpace(row[c])
			if v == "" {
				s.Empty++
				continue
			}
			nonEmpty++
			if !over {
				uniq[v] = struct{}{}
				if len(uniq) > uniqueCap {
					over = true
				}
			}
			if _, err := strconv.ParseInt(v, 10, 64); err == nil {
				ints++
			}
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				floats++
				if first || f < minF {
					minF = f
				}
				if first || f > maxF {
					maxF = f
				}
				first = false
			}
			if len(v) >= 10 && isDate(v[:10]) {
				dates++
				if minD == "" || v < minD {
					minD = v
				}
				if v > maxD {
					maxD = v
				}
			}
		}
		if over {
			s.Unique = ">100000"
		} else {
			s.Unique = strconv.Itoa(len(uniq))
		}
		need := int(float64(nonEmpty)*0.95 + 0.999)
		switch {
		case nonEmpty == 0:
			s.Kind = "text"
		case ints >= need:
			s.Kind, s.Min, s.Max = "int", fmtF(minF), fmtF(maxF)
		case floats >= need:
			s.Kind, s.Min, s.Max = "float", fmtF(minF), fmtF(maxF)
		case dates >= need:
			s.Kind, s.Min, s.Max = "date", minD, maxD
		default:
			s.Kind = "text"
		}
		out[c] = s
	}
	return out
}

func fmtF(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func isDate(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i, r := range s {
		if i != 4 && i != 7 && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// Typed đổi chuỗi thành số/bool nếu parse được (dùng cho --to json --typed).
func Typed(v string) any {
	if i, err := strconv.ParseInt(v, 10, 64); err == nil && !(len(v) > 1 && v[0] == '0') {
		return i
	}
	if f, err := strconv.ParseFloat(v, 64); err == nil && !strings.ContainsAny(v, "eEnNiI") && !(len(v) > 1 && v[0] == '0' && v[1] != '.') {
		return f
	}
	switch strings.ToLower(v) {
	case "true":
		return true
	case "false":
		return false
	}
	return v
}

// Object: một dòng dạng object giữ thứ tự cột.
type Object struct {
	Keys []string
	Vals []any
}

func (o Object) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range o.Keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, err := marshalNoEscape(k)
		if err != nil {
			return nil, err
		}
		vb, err := marshalNoEscape(o.Vals[i])
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func marshalNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := newEncoder(&b)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// Objects đổi bảng thành danh sách object (typed=true thì số/bool thành kiểu thật).
func (t *Table) Objects(typed bool) []Object {
	out := make([]Object, len(t.Rows))
	for i, row := range t.Rows {
		vals := make([]any, len(row))
		for j, v := range row {
			if typed {
				vals[j] = Typed(v)
			} else {
				vals[j] = v
			}
		}
		out[i] = Object{Keys: t.Header, Vals: vals}
	}
	return out
}
