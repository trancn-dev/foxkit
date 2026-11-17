package jwt

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"strings"
	"time"

	"foxkit/internal/ts"
)

var (
	ErrInvalid     = errors.New("JWT không hợp lệ")
	ErrUnsupported = errors.New("JWT không hỗ trợ")
)

type TimeInfo struct {
	Unix     int64  `json:"unix"`
	ISO      string `json:"iso"`
	Relative string `json:"relative"`
}

type Decoded struct {
	Header    map[string]any      `json:"header"`
	Payload   map[string]any      `json:"payload"`
	Signature string              `json:"signature"`
	Times     map[string]TimeInfo `json:"times"`
	Expired   *bool               `json:"expired"`
	Verified  *bool               `json:"verified"`
	Warnings  []string            `json:"warnings"`
}

// Clean bỏ khoảng trắng và tiền tố "Bearer ".
func Clean(token string) string {
	t := strings.TrimSpace(token)
	if len(t) > 7 && strings.EqualFold(t[:7], "bearer ") {
		t = strings.TrimSpace(t[7:])
	}
	return t
}

func decodePart(s string) (map[string]any, error) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(s, "="))
	if err != nil {
		return nil, fmt.Errorf("%w: base64 sai", ErrInvalid)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("%w: phần JSON sai", ErrInvalid)
	}
	return m, nil
}

func split(token string) ([]string, error) {
	parts := strings.Split(Clean(token), ".")
	switch len(parts) {
	case 3:
		return parts, nil
	case 5:
		return nil, fmt.Errorf("%w: JWE (5 phần) chưa hỗ trợ", ErrUnsupported)
	}
	return nil, fmt.Errorf("%w: cần đúng 3 phần xxx.yyy.zzz", ErrInvalid)
}

func unix(v any) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	i, err := n.Int64()
	if err != nil {
		f, ferr := n.Float64()
		return int64(f), ferr == nil
	}
	return i, true
}

// Decode giải mã header/payload và tính hạn dùng theo `now`. Không kiểm chữ ký (dùng VerifyHMAC).
func Decode(token string, now time.Time) (Decoded, error) {
	parts, err := split(token)
	if err != nil {
		return Decoded{}, err
	}
	h, err := decodePart(parts[0])
	if err != nil {
		return Decoded{}, err
	}
	p, err := decodePart(parts[1])
	if err != nil {
		return Decoded{}, err
	}
	d := Decoded{Header: h, Payload: p, Signature: parts[2], Times: map[string]TimeInfo{}, Warnings: []string{}}
	for _, k := range []string{"exp", "iat", "nbf"} {
		if n, ok := unix(p[k]); ok {
			t := time.Unix(n, 0)
			d.Times[k] = TimeInfo{Unix: n, ISO: t.Format(time.RFC3339), Relative: ts.Relative(t, now)}
		}
	}
	if e, ok := d.Times["exp"]; ok {
		expired := now.Unix() > e.Unix
		d.Expired = &expired
	} else {
		d.Warnings = append(d.Warnings, "không có exp (token không hết hạn)")
	}
	if n, ok := d.Times["nbf"]; ok && n.Unix > now.Unix() {
		d.Warnings = append(d.Warnings, "nbf trong tương lai: token chưa có hiệu lực")
	}
	if alg, _ := h["alg"].(string); strings.EqualFold(alg, "none") {
		d.Warnings = append(d.Warnings, "alg=none: token không có chữ ký")
	}
	return d, nil
}

// VerifyHMAC kiểm chữ ký HS256/384/512 trên chuỗi gốc `header.payload`, so bằng hmac.Equal (thời gian hằng).
// Secret dạng `base64:...` (kiểu Laravel) được decode trước.
func VerifyHMAC(token, secret string) (bool, error) {
	parts, err := split(token)
	if err != nil {
		return false, err
	}
	h, err := decodePart(parts[0])
	if err != nil {
		return false, err
	}
	var newHash func() hash.Hash
	switch alg, _ := h["alg"].(string); alg {
	case "HS256":
		newHash = sha256.New
	case "HS384":
		newHash = sha512.New384
	case "HS512":
		newHash = sha512.New
	default:
		return false, fmt.Errorf("%w: chỉ verify HS256/HS384/HS512, token dùng %q", ErrUnsupported, alg)
	}
	key := []byte(secret)
	if raw, ok := strings.CutPrefix(secret, "base64:"); ok {
		b, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return false, fmt.Errorf("%w: secret base64: không giải mã được", ErrInvalid)
		}
		key = b
	}
	mac := hmac.New(newHash, key)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[2], "="))
	if err != nil {
		return false, fmt.Errorf("%w: chữ ký không phải base64url", ErrInvalid)
	}
	return hmac.Equal(want, got), nil
}
