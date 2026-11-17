package textcase

import (
	"fmt"
	"strings"
	"unicode"
)

var Modes = []string{"camel", "studly", "snake", "constant", "kebab", "title", "upper", "lower", "slug"}

// Words tách từ: ranh giới là ký tự không phải chữ/số, chữ thường→hoa, và cuối cụm viết tắt (HTTPServer → HTTP, Server).
func Words(s string) []string {
	rs := []rune(s)
	words := []string{}
	var cur []rune
	flush := func() {
		if len(cur) > 0 {
			words = append(words, string(cur))
			cur = cur[:0]
		}
	}
	for i, r := range rs {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			flush()
			continue
		}
		if len(cur) > 0 && unicode.IsUpper(r) {
			prev := rs[i-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && nextLower) {
				flush()
			}
		}
		cur = append(cur, r)
	}
	flush()
	return words
}

func title(w string) string {
	rs := []rune(strings.ToLower(w))
	if len(rs) > 0 {
		rs[0] = unicode.ToUpper(rs[0])
	}
	return string(rs)
}

// Convert đổi kiểu chữ. camel/studly/snake/constant/kebab/slug đều bỏ dấu; title/upper/lower giữ dấu.
func Convert(mode, s string) (string, error) {
	switch mode {
	case "upper":
		return strings.ToUpper(s), nil
	case "lower":
		return strings.ToLower(s), nil
	case "title":
		return joinMap(Words(s), " ", title), nil
	}
	words := Words(RemoveDiacritics(s))
	switch mode {
	case "camel", "studly", "pascal":
		out := joinMap(words, "", title)
		if mode == "camel" && len(words) > 0 {
			out = strings.ToLower(words[0]) + joinMap(words[1:], "", title)
		}
		return out, nil
	case "snake":
		return joinMap(words, "_", strings.ToLower), nil
	case "constant":
		return joinMap(words, "_", strings.ToUpper), nil
	case "kebab", "slug":
		return joinMap(keepASCII(words, mode == "slug"), "-", strings.ToLower), nil
	}
	return "", fmt.Errorf("kiểu không hỗ trợ: %q", mode)
}

// keepASCII (slug): chỉ giữ a-z0-9, bỏ từ rỗng.
func keepASCII(words []string, on bool) []string {
	if !on {
		return words
	}
	out := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.Map(func(r rune) rune {
			if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
				return r
			}
			return -1
		}, w)
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

func joinMap(words []string, sep string, f func(string) string) string {
	out := make([]string, len(words))
	for i, w := range words {
		out[i] = f(w)
	}
	return strings.Join(out, sep)
}

// All trả mọi kiểu một lượt.
func All(s string) map[string]string {
	m := make(map[string]string, len(Modes))
	for _, mode := range Modes {
		m[mode], _ = Convert(mode, s) // Modes toàn mode hợp lệ nên không lỗi
	}
	return m
}
