package hosts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const defaultHosts = "# Copyright (c) 1993-2009 Microsoft Corp.\r\n#\r\n# For example:\r\n#\r\n#      102.54.94.97     rhino.acme.com          # source server\r\n#       38.25.63.10     x.acme.com              # x client host\r\n\r\n# localhost name resolution is handled within DNS itself.\r\n#\t127.0.0.1       localhost\r\n#\t::1             localhost\r\n"

func TestRoundTripByteForByte(t *testing.T) {
	for _, in := range []string{defaultHosts, "", "127.0.0.1 a.test", "\xef\xbb\xbf127.0.0.1 a.test\n", "127.0.0.1 a b c # x\n\n\n"} {
		if got := string(Parse([]byte(in)).Bytes()); got != in {
			t.Errorf("round-trip lệch:\n%q\n%q", in, got)
		}
	}
	f := Parse([]byte(defaultHosts))
	if len(f.Entries()) != 4 { // 2 dòng ví dụ + 2 dòng localhost, đều bị comment
		t.Errorf("entries = %d", len(f.Entries()))
	}
	for _, e := range f.Entries() {
		if e.Enabled {
			t.Errorf("%+v phải đang tắt", e)
		}
	}
}

func TestAddSkipEnableConflict(t *testing.T) {
	f := Parse([]byte(defaultHosts))
	ch := f.Add("127.0.0.1", []string{"shop.test"})
	if ch[0].Action != "add" || !strings.HasSuffix(string(f.Bytes()), "127.0.0.1 shop.test # foxkit\r\n") {
		t.Fatalf("%+v\n%q", ch, f.Bytes())
	}
	if f.Add("127.0.0.1", []string{"SHOP.test"})[0].Action != "skip" {
		t.Error("lần 2 phải skip (không phân biệt hoa thường)")
	}
	if f.Add("10.0.0.1", []string{"shop.test"})[0].Action != "conflict" {
		t.Error("khác IP phải conflict")
	}
	f.SetEnabled([]string{"shop.test"}, false)
	if got := f.Add("127.0.0.1", []string{"shop.test"})[0].Action; got != "enable" {
		t.Errorf("domain đang tắt phải được bật lại, got %s", got)
	}
}

func TestToggleSplitsMultiDomainLine(t *testing.T) {
	f := Parse([]byte("127.0.0.1 a.test b.test # foxkit\n"))
	ch := f.SetEnabled([]string{"b.test"}, false)
	want := "127.0.0.1 a.test # foxkit\n# 127.0.0.1 b.test # foxkit\n"
	if ch[0].Action != "disable" || string(f.Bytes()) != want {
		t.Fatalf("%+v\n%q", ch, f.Bytes())
	}
	if f.SetEnabled([]string{"zzz.test"}, true)[0].Action != "notfound" {
		t.Error("domain lạ phải notfound")
	}
}

func TestRemove(t *testing.T) {
	f := Parse([]byte("127.0.0.1 a.test b.test\n10.0.0.5 db.internal\n"))
	ch := f.Remove([]string{"b.test", "db.internal", "nope.test"})
	if ch[0].Action != "remove" || ch[2].Action != "notfound" || ch[1].Managed {
		t.Fatalf("%+v", ch)
	}
	if got := string(f.Bytes()); got != "127.0.0.1 a.test\n" {
		t.Fatalf("%q", got)
	}
}

func TestSaveBackupAndAtomic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "hosts")
	if err := os.WriteFile(p, []byte(defaultHosts), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	f.Add("127.0.0.1", []string{"shop.test"})
	bak, err := Save(p, f)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(bak); string(b) != defaultHosts {
		t.Error("backup phải là bản gốc")
	}
	if b, _ := os.ReadFile(p); !strings.Contains(string(b), "shop.test # foxkit") {
		t.Error("file chưa được ghi")
	}
	if _, err := os.Stat(p + ".foxkit.tmp"); err == nil {
		t.Error("file tạm phải được dọn")
	}
}

func TestSaveUnwritableDirReportsPermissionOrError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "no", "such", "dir", "hosts")
	_, err := Save(p, Parse([]byte("127.0.0.1 a\n")))
	if err == nil {
		t.Fatal("cần lỗi")
	}
	if errors.Is(err, os.ErrPermission) {
		t.Log("lỗi quyền được nhận ra")
	}
}

func TestValidators(t *testing.T) {
	for d, want := range map[string]bool{"shop.test": true, "a-b.c": true, "-a.test": false, "a..b": false, "a b": false, "": false} {
		if ValidDomain(d) != want {
			t.Errorf("ValidDomain(%q) != %v", d, want)
		}
	}
	if !ValidIP("::1") || ValidIP("999.1.1.1") {
		t.Error("ValidIP sai")
	}
	t.Setenv("FOXKIT_HOSTS", "X:/h")
	if DefaultPath() != "X:/h" {
		t.Error("FOXKIT_HOSTS phải ghi đè đường dẫn")
	}
}
