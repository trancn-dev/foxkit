package clip

import (
	"errors"
	"os"
	"testing"
)

func TestNormalizeCRLF(t *testing.T) {
	for in, want := range map[string]string{"a\nb": "a\r\nb", "a\r\nb": "a\r\nb", "a\rb": "a\r\nb", "": "", "x\n\n": "x\r\n\r\n"} {
		if got := NormalizeCRLF(in); got != want {
			t.Errorf("NormalizeCRLF(%q) = %q, cần %q", in, got, want)
		}
	}
}

func TestTrimOneNewline(t *testing.T) {
	for in, want := range map[string]string{"abc\n": "abc", "abc\r\n": "abc", "abc\n\n": "abc\n", "abc": "abc", "": ""} {
		if got := TrimOneNewline(in); got != want {
			t.Errorf("TrimOneNewline(%q) = %q, cần %q", in, got, want)
		}
	}
}

func TestRetry(t *testing.T) {
	calls := 0
	err := Retry(func() error {
		calls++
		if calls < 3 {
			return errors.New("bận")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Fatalf("%v %d", err, calls)
	}
	if err := Retry(func() error { return errors.New("luôn bận") }); err == nil {
		t.Fatal("hết 3 lần phải trả lỗi")
	}
}

// Test thật ghi đè clipboard của người dùng nên chỉ chạy khi FOXKIT_TEST_CLIPBOARD=1.
func TestRealClipboardRoundTrip(t *testing.T) {
	if os.Getenv("FOXKIT_TEST_CLIPBOARD") != "1" {
		t.Skip("đặt FOXKIT_TEST_CLIPBOARD=1 để chạy (sẽ ghi đè clipboard)")
	}
	const s = "Xin chào đ 🚀"
	if err := Default.Write(s); err != nil {
		t.Fatal(err)
	}
	if got, err := Default.Read(); err != nil || got != s {
		t.Fatalf("%q %v", got, err)
	}
}
