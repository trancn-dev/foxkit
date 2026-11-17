package disk

import (
	"runtime"
	"strings"
	"testing"
)

func TestSplitMultiSZ(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"C:\\\x00D:\\\x00\x00", `C:\|D:\`},
		{"\x00\x00", ""},
		{"", ""},
		{"C:\\", `C:\`},
	}
	for _, tc := range tests {
		if got := strings.Join(SplitMultiSZ(tc.in), "|"); got != tc.want {
			t.Errorf("SplitMultiSZ(%q) = %q, cần %q", tc.in, got, tc.want)
		}
	}
}

func TestFillUsedPctAndWarn(t *testing.T) {
	d := Drive{TotalBytes: 1000, FreeBytes: 80}
	d.Fill(90)
	if d.UsedPct != 92 || !d.Warn {
		t.Errorf("%+v", d)
	}
	d = Drive{TotalBytes: 1000, FreeBytes: 550}
	d.Fill(90)
	if d.UsedPct != 45 || d.Warn {
		t.Errorf("%+v", d)
	}
	d = Drive{}
	d.Fill(90) // ổ rỗng/lỗi không chia cho 0
	if d.UsedPct != 0 || d.Warn {
		t.Errorf("%+v", d)
	}
}

func TestTypeName(t *testing.T) {
	for code, want := range map[uint32]string{2: "removable", 3: "fixed", 4: "network", 5: "cdrom", 6: "ram", 0: "unknown", 1: "unknown"} {
		if TypeName(code) != want {
			t.Errorf("TypeName(%d) != %s", code, want)
		}
	}
}

func TestListRealDrives(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ Windows")
	}
	ds, err := List(false, 90)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range ds {
		if strings.EqualFold(d.Root, `C:\`) {
			if d.TotalBytes == 0 || d.Type != "fixed" || d.FS == "" {
				t.Fatalf("%+v", d)
			}
			return
		}
	}
	t.Fatalf("không thấy ổ C trong %+v", ds)
}
