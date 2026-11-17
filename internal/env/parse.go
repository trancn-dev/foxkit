package env

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// Line là một dòng logic của file .env (giá trị nhiều dòng trong nháy kép gộp thành một Line).
type Line struct {
	Raw   string // nguyên văn kèm ký tự xuống dòng, để ghi lại không đổi byte
	Key   string // rỗng nếu là comment/dòng trống
	Value string // đã bỏ nháy, đã xử lý escape trong nháy kép
	Line  int    // số dòng bắt đầu (1-based)
}

type File struct{ Lines []Line }

// ParseError mang số dòng để cmd báo hint "Kiểm tra dòng N".
type ParseError struct {
	Line int
	Msg  string
}

func (e *ParseError) Error() string { return fmt.Sprintf("dòng %d: %s", e.Line, e.Msg) }

var keyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*$`)

func ParseFile(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(strings.NewReader(string(b)))
}

// splitKeep tách theo "\n" nhưng giữ lại terminator ("\n" hoặc "\r\n") trong từng phần tử.
func splitKeep(s string) []string {
	var out []string
	for len(s) > 0 {
		i := strings.IndexByte(s, '\n')
		if i < 0 {
			out = append(out, s)
			break
		}
		out = append(out, s[:i+1])
		s = s[i+1:]
	}
	return out
}

func Parse(r io.Reader) (*File, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	phys := splitKeep(string(data))
	f := &File{Lines: []Line{}}
	for i := 0; i < len(phys); {
		start := i + 1
		raw := phys[i]
		i++
		t := strings.TrimSpace(raw)
		if t == "" || strings.HasPrefix(t, "#") {
			f.Lines = append(f.Lines, Line{Raw: raw, Line: start})
			continue
		}
		t = strings.TrimSpace(strings.TrimPrefix(t, "export "))
		eq := strings.Index(t, "=")
		if eq < 0 {
			return nil, &ParseError{start, "thiếu dấu ="}
		}
		key := strings.TrimSpace(t[:eq])
		if !keyRe.MatchString(key) {
			return nil, &ParseError{start, fmt.Sprintf("tên biến không hợp lệ %q", key)}
		}
		rest := strings.TrimLeft(t[eq+1:], " \t")
		var value string
		switch {
		case strings.HasPrefix(rest, `"`):
			for {
				v, ok := unquoteDouble(rest[1:])
				if ok {
					value = v
					break
				}
				if i >= len(phys) {
					return nil, &ParseError{start, "thiếu dấu \" đóng"}
				}
				rest += "\n" + strings.TrimRight(phys[i], "\r\n")
				raw += phys[i]
				i++
			}
		case strings.HasPrefix(rest, "'"):
			end := strings.Index(rest[1:], "'")
			if end < 0 {
				return nil, &ParseError{start, "thiếu dấu ' đóng"}
			}
			value = rest[1 : 1+end]
		default:
			cut := len(rest)
			if strings.HasPrefix(rest, "#") {
				cut = 0
			}
			for _, sep := range []string{" #", "\t#"} { // "#" dính liền giá trị (a#b) không phải comment
				if j := strings.Index(rest, sep); j >= 0 && j < cut {
					cut = j
				}
			}
			rest = rest[:cut]
			value = strings.TrimSpace(rest)
		}
		f.Lines = append(f.Lines, Line{Raw: raw, Key: key, Value: value, Line: start})
	}
	return f, nil
}

// unquoteDouble đọc tới dấu " đóng không bị escape; ok=false nếu chưa thấy.
func unquoteDouble(s string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\\':
			if i+1 >= len(s) {
				return "", false
			}
			i++
			switch s[i] {
			case 'n':
				b.WriteByte('\n')
			case '"':
				b.WriteByte('"')
			case '\\':
				b.WriteByte('\\')
			default:
				b.WriteByte('\\')
				b.WriteByte(s[i])
			}
		case '"':
			return b.String(), true
		default:
			b.WriteByte(c)
		}
	}
	return "", false
}

// Get trả giá trị cuối cùng của key (giống phpdotenv khi trùng key).
func (f *File) Get(key string) (string, bool) {
	for i := len(f.Lines) - 1; i >= 0; i-- {
		if f.Lines[i].Key == key {
			return f.Lines[i].Value, true
		}
	}
	return "", false
}

// LineOf trả số dòng của lần khai báo cuối cùng, 0 nếu không có.
func (f *File) LineOf(key string) int {
	for i := len(f.Lines) - 1; i >= 0; i-- {
		if f.Lines[i].Key == key {
			return f.Lines[i].Line
		}
	}
	return 0
}

// Keys theo thứ tự xuất hiện lần đầu, không trùng.
func (f *File) Keys() []string {
	keys, seen := []string{}, map[string]bool{}
	for _, l := range f.Lines {
		if l.Key != "" && !seen[l.Key] {
			seen[l.Key] = true
			keys = append(keys, l.Key)
		}
	}
	return keys
}

// Duplicates: các key khai báo nhiều lần.
func (f *File) Duplicates() []string {
	dups, count := []string{}, map[string]int{}
	for _, l := range f.Lines {
		if l.Key == "" {
			continue
		}
		if count[l.Key]++; count[l.Key] == 2 {
			dups = append(dups, l.Key)
		}
	}
	return dups
}

func needsQuote(v string) bool { return strings.ContainsAny(v, " \t#\"'\\\n$") }

func formatValue(v string) string {
	if !needsQuote(v) {
		return v
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(v) + `"`
}

// Set sửa đúng dòng cũ (lần khai báo cuối) hoặc thêm cuối file; các dòng khác giữ nguyên byte.
func (f *File) Set(key, value string) {
	nl := "\n"
	if len(f.Lines) > 0 && strings.HasSuffix(f.Lines[0].Raw, "\r\n") {
		nl = "\r\n"
	}
	text := key + "=" + formatValue(value)
	for i := len(f.Lines) - 1; i >= 0; i-- {
		if f.Lines[i].Key == key {
			term := ""
			if r := f.Lines[i].Raw; strings.HasSuffix(r, "\r\n") {
				term = "\r\n"
			} else if strings.HasSuffix(r, "\n") {
				term = "\n"
			}
			f.Lines[i].Raw, f.Lines[i].Value = text+term, value
			return
		}
	}
	if n := len(f.Lines); n > 0 && !strings.HasSuffix(f.Lines[n-1].Raw, "\n") {
		f.Lines[n-1].Raw += nl
	}
	f.Lines = append(f.Lines, Line{Raw: text + nl, Key: key, Value: value, Line: len(f.Lines) + 1})
}

func (f *File) WriteTo(w io.Writer) (int64, error) {
	var total int64
	for _, l := range f.Lines {
		n, err := io.WriteString(w, l.Raw)
		total += int64(n)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}
