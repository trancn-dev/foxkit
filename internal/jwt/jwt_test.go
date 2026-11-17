package jwt

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Token mẫu của jwt.io: HS256, secret "your-256-bit-secret", iat 1516239022.
const sample = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

var now = time.Unix(1700000000, 0)

func TestVerifyJwtIoVector(t *testing.T) {
	if ok, err := VerifyHMAC(sample, "your-256-bit-secret"); err != nil || !ok {
		t.Fatalf("vector jwt.io phải hợp lệ: %v %v", ok, err)
	}
	if ok, _ := VerifyHMAC(sample, "sai-secret"); ok {
		t.Error("secret sai phải false")
	}
	tampered := strings.Replace(sample, "IkpvaG4", "IkpvaG5", 1)
	if tampered == sample {
		tampered = sample[:50] + "X" + sample[51:]
	}
	if ok, err := VerifyHMAC(tampered, "your-256-bit-secret"); ok && err == nil {
		t.Error("đổi payload phải false")
	}
	if ok, err := VerifyHMAC("Bearer "+sample, "your-256-bit-secret"); err != nil || !ok {
		t.Errorf("Bearer phải được bỏ: %v", err)
	}
}

func TestDecode(t *testing.T) {
	d, err := Decode(sample, now)
	if err != nil {
		t.Fatal(err)
	}
	if d.Header["alg"] != "HS256" || d.Payload["name"] != "John Doe" {
		t.Fatalf("%+v", d)
	}
	if d.Times["iat"].Unix != 1516239022 || d.Expired != nil || d.Verified != nil {
		t.Fatalf("%+v", d)
	}
	if len(d.Warnings) != 1 || !strings.Contains(d.Warnings[0], "exp") {
		t.Errorf("phải cảnh báo thiếu exp: %v", d.Warnings)
	}
}

func TestExpiry(t *testing.T) {
	// payload {"exp":1700000100}
	tok := "eyJhbGciOiJIUzI1NiJ9.eyJleHAiOjE3MDAwMDAxMDB9.x"
	d, err := Decode(tok, now)
	if err != nil || d.Expired == nil || *d.Expired {
		t.Fatalf("còn hạn: %+v %v", d, err)
	}
	d, _ = Decode(tok, time.Unix(1700000200, 0))
	if !*d.Expired || d.Times["exp"].Relative != "3 phút trước" && !strings.Contains(d.Times["exp"].Relative, "trước") {
		t.Fatalf("hết hạn: %+v", d)
	}
}

func TestAlgNoneWarningAndUnsupported(t *testing.T) {
	// header {"alg":"none"}, payload {}
	tok := "eyJhbGciOiJub25lIn0.e30."
	d, err := Decode(tok, now)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, w := range d.Warnings {
		if strings.Contains(w, "alg=none") {
			found = true
		}
	}
	if !found {
		t.Errorf("thiếu cảnh báo alg=none: %v", d.Warnings)
	}
	if _, err := VerifyHMAC(tok, "x"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("alg none không verify được: %v", err)
	}
}

func TestInvalidInputs(t *testing.T) {
	for _, bad := range []string{"", "a.b", "a.b.c.d", "!!!.!!!.!!!"} {
		if _, err := Decode(bad, now); !errors.Is(err, ErrInvalid) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if _, err := Decode("a.b.c.d.e", now); !errors.Is(err, ErrUnsupported) {
		t.Error("JWE phải là unsupported")
	}
}

func TestBase64Secret(t *testing.T) {
	// HS256 với secret "k" (base64: azk=... = "k" → "aw==")
	if ok, err := VerifyHMAC(sample, "base64:!!"); ok || !errors.Is(err, ErrInvalid) {
		t.Errorf("secret base64 hỏng phải báo lỗi: %v %v", ok, err)
	}
}
