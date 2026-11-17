package wifi

import (
	"os"
	"strings"
	"testing"
	"unicode/utf16"
)

func parse(t *testing.T, name string) Profile {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProfileXML(b)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseProfiles(t *testing.T) {
	p := parse(t, "profile_wpa2.xml")
	if p.SSID != "Office-5G" || p.Auth != "WPA2PSK" || p.Password != "fake-pass-for-test" || p.Hidden || p.Error != "" {
		t.Errorf("wpa2: %+v", p)
	}
	o := parse(t, "profile_open.xml")
	if o.SSID != "Cafe Free" || o.Auth != "open" || o.Password != "" || !o.Hidden {
		t.Errorf("open: %+v", o)
	}
	pr := parse(t, "profile_protected.xml")
	if pr.Password != "" || pr.Error != "cần quyền admin để xem mật khẩu" || pr.Auth != "WPA3SAE" {
		t.Errorf("protected: không được lộ keyMaterial đã mã hóa: %+v", pr)
	}
}

func TestParseUTF16WithBOM(t *testing.T) {
	src, _ := os.ReadFile("testdata/profile_wpa2.xml")
	u := utf16.Encode([]rune(strings.Replace(string(src), `<?xml version="1.0"?>`, `<?xml version="1.0" encoding="utf-16"?>`, 1)))
	b := []byte{0xFF, 0xFE}
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	p, err := ParseProfileXML(b)
	if err != nil || p.SSID != "Office-5G" {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestBadXML(t *testing.T) {
	if _, err := ParseProfileXML([]byte("không phải xml")); err == nil {
		t.Fatal("cần lỗi")
	}
}
