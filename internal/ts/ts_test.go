package ts

import (
	"testing"
	"time"
)

var now = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func TestParse(t *testing.T) {
	hcm, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		t.Skip("không có tzdata:", err)
	}
	tests := []struct {
		name, value string
		loc         *time.Location
		wantUTC     string
		wantKind    string
		wantErr     bool
	}{
		{"zero", "0", time.UTC, "1970-01-01T00:00:00Z", "seconds", false},
		{"ms", "1763287727000", time.UTC, "2025-11-16T10:08:47Z", "milliseconds", false},
		{"datetime hcm", "2026-11-16 19:08:47", hcm, "2026-11-16T12:08:47Z", "datetime", false},
		{"rfc3339 offset", "2026-11-16T19:08:47+07:00", time.UTC, "2026-11-16T12:08:47Z", "rfc3339", false},
		{"date", "2026-11-16", hcm, "2026-11-15T17:00:00Z", "date", false},
		{"now", "now", time.UTC, "2026-11-16T12:00:00Z", "now", false},
		{"empty", "", time.UTC, "2026-11-16T12:00:00Z", "now", false},
		{"garbage", "abc", time.UTC, "", "", true},
		{"overflow", "9999999999999999999", time.UTC, "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, kind, err := Parse(tc.value, tc.loc, now)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v", err)
			}
			if tc.wantErr {
				return
			}
			if u := got.UTC().Format(time.RFC3339); u != tc.wantUTC || kind != tc.wantKind {
				t.Fatalf("got %s %s", u, kind)
			}
		})
	}
}

func TestRelative(t *testing.T) {
	tests := []struct {
		t    time.Time
		want string
	}{
		{now.Add(-3 * time.Hour), "3 giờ trước"},
		{now.Add(48 * time.Hour), "sau 2 ngày"},
		{now.Add(-2 * time.Second), "vừa xong"},
		{now.Add(-90 * time.Minute), "1 giờ trước"},
		{now.Add(-400 * 24 * time.Hour), "1 năm trước"},
		{now.Add(-45 * 24 * time.Hour), "1 tháng trước"},
	}
	for _, tc := range tests {
		if got := Relative(tc.t, now); got != tc.want {
			t.Errorf("Relative = %q want %q", got, tc.want)
		}
	}
}

func TestDescribeWeekday(t *testing.T) {
	// 2026-11-16 là Thứ Hai.
	if got := Describe(now, time.UTC, now); got.Weekday != "Thứ Hai" || got.Zone != "UTC (+00:00)" {
		t.Fatalf("got %+v", got)
	}
}
