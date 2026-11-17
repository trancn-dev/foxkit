package env

import "strings"

type Diff struct {
	Missing []string `json:"missing"` // có ở example, không có ở .env
	Extra   []string `json:"extra"`   // có ở .env, không có ở example
	Empty   []string `json:"empty"`   // có ở .env nhưng rỗng, trong khi example có giá trị
}

// Compare: Missing/Empty theo thứ tự trong example, Extra theo thứ tự trong .env.
func Compare(env, example *File) Diff {
	d := Diff{Missing: []string{}, Extra: []string{}, Empty: []string{}}
	inEnv, inExample := map[string]bool{}, map[string]bool{}
	for _, k := range env.Keys() {
		inEnv[k] = true
	}
	for _, k := range example.Keys() {
		inExample[k] = true
		if !inEnv[k] {
			d.Missing = append(d.Missing, k)
			continue
		}
		ev, _ := env.Get(k)
		xv, _ := example.Get(k)
		if ev == "" && xv != "" {
			d.Empty = append(d.Empty, k)
		}
	}
	for _, k := range env.Keys() {
		if !inExample[k] {
			d.Extra = append(d.Extra, k)
		}
	}
	return d
}

var secretWords = []string{"KEY", "SECRET", "PASSWORD", "PASS", "TOKEN", "PRIVATE"}

func IsSecretKey(key string) bool {
	k := strings.ToUpper(key)
	for _, w := range secretWords {
		if strings.Contains(k, w) {
			return true
		}
	}
	return false
}

// Mask che giá trị bí mật: 3 ký tự đầu + "***", hoặc "***" nếu ngắn hơn 8 ký tự.
func Mask(v string) string {
	r := []rune(v)
	if len(r) < 8 {
		return "***"
	}
	return string(r[:3]) + "***"
}
