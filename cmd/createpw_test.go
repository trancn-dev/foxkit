package cmd

import (
	"strings"
	"testing"
)

func TestCreatepw(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		check    func(t *testing.T, out string)
	}{
		{"mặc định 16 ký tự, bắt đầu bằng chữ hoa", []string{"createpw"}, 0, func(t *testing.T, out string) {
			pw := strings.TrimSpace(out)
			if len(pw) != 16 || pw[0] < 'A' || pw[0] > 'Z' {
				t.Errorf("mật khẩu %q: muốn 16 ký tự, đầu là chữ hoa", pw)
			}
		}},
		{"--len 20 --count 3", []string{"createpw", "--len", "20", "--count", "3"}, 0, func(t *testing.T, out string) {
			lines := strings.Split(strings.TrimSpace(out), "\n")
			if len(lines) != 3 || len(lines[0]) != 20 {
				t.Errorf("muốn 3 dòng 20 ký tự, có %q", out)
			}
		}},
		{"độ dài ngoài khoảng → exit 2", []string{"createpw", "--len", "3"}, 2, nil},
		{"số lượng ngoài khoảng → exit 2", []string{"createpw", "--count", "101"}, 2, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, _, code := execute(tc.args...)
			if code != tc.wantCode {
				t.Fatalf("exit %d, muốn %d", code, tc.wantCode)
			}
			if tc.check != nil {
				tc.check(t, out)
			}
		})
	}
}

// --copy ghi mật khẩu ĐẦU TIÊN vào clipboard (dùng clipboard giả, không đụng clipboard thật).
func TestCreatepwCopy(t *testing.T) {
	f := withFake(t)
	out, errOut, code := execute("createpw", "--copy", "--count", "2")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	first := strings.Split(strings.TrimSpace(out), "\n")[0]
	if f.text != first {
		t.Errorf("clipboard %q ≠ mật khẩu đầu %q", f.text, first)
	}
	if !strings.Contains(errOut, "Đã copy") {
		t.Errorf("thiếu thông báo copy trên stderr: %q", errOut)
	}
}

func TestCreatepwCopyFailure(t *testing.T) {
	f := withFake(t)
	f.fail = true
	if _, _, code := execute("createpw", "--copy"); code != 1 {
		t.Errorf("clipboard lỗi → exit %d, muốn 1", code)
	}
}
