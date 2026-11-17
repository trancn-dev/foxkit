package jsonx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type SyntaxErr struct {
	Line, Col int
	Msg       string
	Context   string // đoạn quanh chỗ lỗi + dòng thứ hai có dấu ^
}

func (e *SyntaxErr) Error() string {
	return fmt.Sprintf("JSON không hợp lệ ở dòng %d, cột %d: %s", e.Line, e.Col, e.Msg)
}

// StripBOM bỏ BOM UTF-8 đầu file (file JSON tạo trên Windows hay có).
func StripBOM(b []byte) []byte { return bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")) }

// Validate trả nil nếu hợp lệ; ngược lại vị trí dòng/cột (1-based) và đoạn ngữ cảnh.
func Validate(b []byte) *SyntaxErr {
	b = StripBOM(b)
	if json.Valid(b) {
		return nil
	}
	var v any
	err := json.Unmarshal(b, &v)
	var se *json.SyntaxError
	if !errors.As(err, &se) {
		return &SyntaxErr{Line: 1, Col: 1, Msg: fmt.Sprint(err)}
	}
	idx := int(se.Offset) - 1 // Offset = số byte đã đọc, ký tự lỗi là byte cuối cùng đó
	if idx < 0 {
		idx = 0
	}
	if idx > len(b) {
		idx = len(b)
	}
	line := bytes.Count(b[:idx], []byte("\n")) + 1
	col := idx - (bytes.LastIndexByte(b[:idx], '\n') + 1) + 1
	return &SyntaxErr{Line: line, Col: col, Msg: se.Error(), Context: context(b, idx, col)}
}

// context: dòng chứa lỗi (cắt ±40 ký tự) và dòng có ^ chỉ đúng cột.
func context(b []byte, idx, col int) string {
	start := bytes.LastIndexByte(b[:idx], '\n') + 1
	end := bytes.IndexByte(b[idx:], '\n')
	if end < 0 {
		end = len(b)
	} else {
		end += idx
	}
	line := []rune(strings.TrimRight(string(b[start:end]), "\r"))
	pos := len([]rune(string(b[start:idx])))
	from := max(0, pos-40)
	to := min(len(line), pos+40)
	if pos >= len(line) {
		to = len(line)
	}
	return string(line[from:to]) + "\n" + strings.Repeat(" ", pos-from) + "^"
}

// Pretty / Minify dùng json.Indent / json.Compact trên byte gốc: giữ thứ tự key và giữ nguyên số (1.0, ID 19 chữ số).
func Pretty(b []byte, indent string) ([]byte, error) {
	var out bytes.Buffer
	if err := json.Indent(&out, StripBOM(b), "", indent); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func Minify(b []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := json.Compact(&out, StripBOM(b)); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func decode(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(StripBOM(b)))
	dec.UseNumber() // giữ số chính xác
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// Encode ghi giá trị ra JSON không escape HTML; indent rỗng = gọn.
func Encode(v any, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// SortKeys: decode UseNumber rồi encode lại (map được Go tự sắp key theo ABC).
func SortKeys(b []byte, indent string) ([]byte, error) {
	v, err := decode(b)
	if err != nil {
		return nil, err
	}
	return Encode(v, indent)
}

// PathError: cú pháp đường dẫn sai.
type PathError struct{ Msg string }

func (e *PathError) Error() string { return e.Msg }

// NotFound: Walked là phần đường dẫn đã đi được, Missing là bước không có.
type NotFound struct{ Walked, Missing string }

func (e *NotFound) Error() string {
	return fmt.Sprintf("Đã đi tới %q, không có %q", e.Walked, e.Missing)
}

// ParsePath tách `a.b.0`, `["key.with.dot"]`, `items.#`, `items.*.id`.
func ParsePath(p string) ([]string, error) {
	var toks []string
	for i := 0; i < len(p); {
		switch p[i] {
		case '.':
			i++
		case '[':
			if i+1 >= len(p) || p[i+1] != '"' {
				return nil, &PathError{`Sau "[" phải là chuỗi trong nháy kép, ví dụ ["a.b"]`}
			}
			j := i + 2
			for j < len(p) && !(p[j] == '"' && p[j-1] != '\\') {
				j++
			}
			if j+1 >= len(p) || p[j+1] != ']' {
				return nil, &PathError{"Thiếu dấu \"] ở đường dẫn"}
			}
			s, err := strconv.Unquote(p[i+1 : j+1])
			if err != nil {
				return nil, &PathError{"Chuỗi trong [\"...\"] không hợp lệ"}
			}
			toks = append(toks, s)
			i = j + 2
		default:
			j := i
			for j < len(p) && p[j] != '.' && p[j] != '[' {
				j++
			}
			toks = append(toks, p[i:j])
			i = j
		}
	}
	if len(toks) == 0 {
		return nil, &PathError{"Đường dẫn rỗng"}
	}
	return toks, nil
}

// Get lấy giá trị theo đường dẫn. `#` = độ dài mảng/object, `*` = áp phần còn lại cho mọi phần tử, số âm = đếm từ cuối.
func Get(b []byte, path string) (any, error) {
	toks, err := ParsePath(path)
	if err != nil {
		return nil, err
	}
	v, err := decode(b)
	if err != nil {
		return nil, err
	}
	return walk(v, toks, "")
}

func join(walked, tok string) string {
	if walked == "" {
		return tok
	}
	return walked + "." + tok
}

func walk(v any, toks []string, walked string) (any, error) {
	if len(toks) == 0 {
		return v, nil
	}
	tok, rest := toks[0], toks[1:]
	switch x := v.(type) {
	case map[string]any:
		if tok == "#" {
			return walk(json.Number(strconv.Itoa(len(x))), rest, join(walked, tok))
		}
		if tok == "*" {
			out := make([]any, 0, len(x))
			for _, k := range sortedKeys(x) {
				r, err := walk(x[k], rest, join(walked, k))
				if err == nil {
					out = append(out, r)
				}
			}
			return out, nil
		}
		if c, ok := x[tok]; ok {
			return walk(c, rest, join(walked, tok))
		}
	case []any:
		switch tok {
		case "#":
			return walk(json.Number(strconv.Itoa(len(x))), rest, join(walked, tok))
		case "*":
			out := make([]any, 0, len(x))
			for i, c := range x {
				if r, err := walk(c, rest, join(walked, strconv.Itoa(i))); err == nil { // phần tử thiếu key thì bỏ qua
					out = append(out, r)
				}
			}
			return out, nil
		}
		if i, err := strconv.Atoi(tok); err == nil {
			if i < 0 {
				i += len(x)
			}
			if i >= 0 && i < len(x) {
				return walk(x[i], rest, join(walked, tok))
			}
		}
	}
	return nil, &NotFound{Walked: walked, Missing: tok}
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// sắp ABC để kết quả ổn định (map của Go không có thứ tự)
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
