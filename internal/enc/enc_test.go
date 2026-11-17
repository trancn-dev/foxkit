package enc

import (
	"bytes"
	"testing"
)

func TestFixed(t *testing.T) {
	tests := []struct{ mode, in, want string }{
		{"base64", "hello", "aGVsbG8="},
		{"url", "a b&c", "a+b%26c"},
		{"html", "<a>", "&lt;a&gt;"},
		{"hex", "hi", "6869"},
		{"base64url", "\xfb\xff", "-_8"},
	}
	for _, tc := range tests {
		got, err := Encode(tc.mode, []byte(tc.in))
		if err != nil || got != tc.want {
			t.Errorf("Encode(%s,%q)=%q,%v want %q", tc.mode, tc.in, got, err, tc.want)
		}
	}
}

func TestDecodeLenient(t *testing.T) {
	tests := []struct{ mode, in, want string }{
		{"base64", "aGVsbG8", "hello"},     // thiếu padding
		{"base64", " aGVsbG8=\n", "hello"}, // khoảng trắng thừa khi pipe
		{"base64url", "aGVsbG8=", "hello"}, // có padding
		{"hex", "68 69", "hi"},             // khoảng trắng trong hex
	}
	for _, tc := range tests {
		got, err := Decode(tc.mode, tc.in)
		if err != nil || string(got) != tc.want {
			t.Errorf("Decode(%s,%q)=%q,%v", tc.mode, tc.in, got, err)
		}
	}
	if _, err := Decode("base64", "!!!"); err == nil {
		t.Error("cần lỗi")
	}
	if _, err := Decode("zz", "x"); err == nil {
		t.Error("cần lỗi mode")
	}
}

func TestRoundTrip(t *testing.T) {
	all := make([]byte, 256)
	for i := range all {
		all[i] = byte(i)
	}
	inputs := [][]byte{{}, []byte("hello"), []byte("Xin chào đ"), all}
	for _, mode := range []string{"base64", "base64url", "hex"} {
		for _, in := range inputs {
			e, err := Encode(mode, in)
			if err != nil {
				t.Fatal(err)
			}
			d, err := Decode(mode, e)
			if err != nil || !bytes.Equal(d, in) {
				t.Errorf("%s round-trip %q lỗi: %v", mode, in, err)
			}
		}
	}
	for _, mode := range []string{"url", "html"} {
		for _, in := range inputs[:3] {
			e, _ := Encode(mode, in)
			d, err := Decode(mode, e)
			if err != nil || !bytes.Equal(d, in) {
				t.Errorf("%s round-trip %q lỗi: %v", mode, in, err)
			}
		}
	}
}
