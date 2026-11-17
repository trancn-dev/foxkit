package pass

import (
	"crypto/rand"
	"fmt"
	"math"
	"math/big"
	"strings"
)

type Options struct {
	Length      int
	Charset     string
	NoAmbiguous bool
}

const (
	lower   = "abcdefghijklmnopqrstuvwxyz"
	upper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	digits  = "0123456789"
	symbols = "!@#$%^&*()-_=+[]{};:,.?"
	// ambiguous: ký tự dễ nhầm với nhau.
	ambiguous = "0Oo1lI|"
)

// Alphabet trả bộ ký tự theo charset, đã lọc ký tự dễ nhầm nếu cần.
func Alphabet(charset string, noAmbiguous bool) (string, error) {
	var a string
	switch charset {
	case "all":
		a = lower + upper + digits + symbols
	case "alnum":
		a = lower + upper + digits
	case "alpha":
		a = lower + upper
	case "num":
		a = digits
	case "hex":
		a = digits + "abcdef"
	default:
		return "", fmt.Errorf("charset không hợp lệ: %q", charset)
	}
	if noAmbiguous {
		a = strings.Map(func(r rune) rune {
			if strings.ContainsRune(ambiguous, r) {
				return -1
			}
			return r
		}, a)
	}
	return a, nil
}

// EntropyBits = length * log2(alphabetSize).
func EntropyBits(length, alphabetSize int) float64 {
	return float64(length) * math.Log2(float64(alphabetSize))
}

// Generate sinh một chuỗi ngẫu nhiên. Với charset "all" đảm bảo có đủ 4 loại ký tự.
func Generate(opt Options) (string, error) {
	alpha, err := Alphabet(opt.Charset, opt.NoAmbiguous)
	if err != nil {
		return "", err
	}
	max := big.NewInt(int64(len(alpha)))
	need4 := opt.Charset == "all" && opt.Length >= 4
	// len=4 chỉ ~6% chuỗi đủ 4 loại → 100 lần không đủ; 10000 lần thì xác suất thất bại ≈ 0.
	for try := 0; try < 10000; try++ {
		var b strings.Builder
		for i := 0; i < opt.Length; i++ {
			n, err := rand.Int(rand.Reader, max) // không lệch phân bố như byte % len
			if err != nil {
				return "", err
			}
			b.WriteByte(alpha[n.Int64()])
		}
		s := b.String()
		if !need4 || hasAllClasses(s) {
			return s, nil
		}
	}
	return "", fmt.Errorf("không sinh được mật khẩu đủ 4 loại ký tự sau 10000 lần")
}

func hasAllClasses(s string) bool {
	return strings.ContainsAny(s, lower) && strings.ContainsAny(s, upper) &&
		strings.ContainsAny(s, digits) && strings.ContainsAny(s, symbols)
}
