package textcase

import "strings"

// diacritics: mỗi dòng "gốc" + các biến thể có dấu (thường rồi hoa).
var diacritics = map[rune]rune{}

func init() {
	groups := map[rune]string{
		'a': "àáạảãâầấậẩẫăằắặẳẵ",
		'e': "èéẹẻẽêềếệểễ",
		'i': "ìíịỉĩ",
		'o': "òóọỏõôồốộổỗơờớợởỡ",
		'u': "ùúụủũưừứựửữ",
		'y': "ỳýỵỷỹ",
		'd': "đ",
	}
	for base, chars := range groups {
		for _, r := range chars {
			diacritics[r] = base
			diacritics[[]rune(strings.ToUpper(string(r)))[0]] = []rune(strings.ToUpper(string(base)))[0]
		}
	}
}

// RemoveDiacritics: "Đơn hàng" → "Don hang".
func RemoveDiacritics(s string) string {
	return strings.Map(func(r rune) rune {
		if b, ok := diacritics[r]; ok {
			return b
		}
		return r
	}, s)
}
