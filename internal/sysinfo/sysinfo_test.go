package sysinfo

import (
	"runtime"
	"testing"
	"time"
)

func TestFormatUptime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{59 * time.Second, "59 giây"},
		{3*time.Hour + 5*time.Minute, "3 giờ 5 phút"},
		{3*24*time.Hour + 4*time.Hour + 12*time.Minute, "3 ngày 4 giờ 12 phút"},
		{0, "0 giây"},
		{2 * 24 * time.Hour, "2 ngày"},
	}
	for _, tc := range tests {
		if got := FormatUptime(tc.d); got != tc.want {
			t.Errorf("FormatUptime(%v) = %q, cần %q", tc.d, got, tc.want)
		}
	}
}

func TestFixWindowsName(t *testing.T) {
	tests := []struct {
		name  string
		build int
		want  string
	}{
		{"Windows 10 Pro", 26100, "Windows 11 Pro"},
		{"Windows 10 Pro", 22000, "Windows 11 Pro"},
		{"Windows 10 Pro", 19045, "Windows 10 Pro"},
		{"Windows Server 2022 Standard", 20348, "Windows Server 2022 Standard"},
	}
	for _, tc := range tests {
		if got := FixWindowsName(tc.name, tc.build); got != tc.want {
			t.Errorf("FixWindowsName(%q,%d) = %q", tc.name, tc.build, got)
		}
	}
}

func TestMachineArch(t *testing.T) {
	for in, want := range map[string]string{"AMD64": "amd64", "ARM64": "arm64", "x86": "386", "": ""} {
		if got := MachineArch(in); got != want {
			t.Errorf("MachineArch(%q) = %q", in, got)
		}
	}
}

func TestGetOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ Windows")
	}
	in, err := Get()
	if err != nil {
		t.Fatal(err)
	}
	if in.Cores == 0 || in.RAMBytes == 0 || in.OS == "" || in.Hostname == "" || in.UptimeSec <= 0 {
		t.Fatalf("%+v", in)
	}
}
