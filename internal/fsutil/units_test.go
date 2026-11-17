package fsutil

import (
	"testing"
	"time"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		in   string
		want int64
		ok   bool
	}{
		{"500", 500, true}, {"1KB", 1024, true}, {"10mb", 10 << 20, true}, {"2GB", 2 << 30, true},
		{"1.5KB", 1536, true}, {"", 0, false}, {"abc", 0, false}, {"-1KB", 0, false}, {"1XB", 0, false},
		{"NaN", 0, false}, {"inf", 0, false},
	}
	for _, tc := range tests {
		got, err := ParseSize(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseSize(%q)=%d,%v", tc.in, got, err)
		}
	}
}

func TestParseAge(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"90m", 90 * time.Minute, true}, {"12h", 12 * time.Hour, true}, {"30d", 30 * 24 * time.Hour, true},
		{"2w", 14 * 24 * time.Hour, true}, {"1y", 365 * 24 * time.Hour, true}, {"x", 0, false}, {"5", 0, false}, {"dd", 0, false},
	}
	for _, tc := range tests {
		got, err := ParseAge(tc.in)
		if (err == nil) != tc.ok || got != tc.want {
			t.Errorf("ParseAge(%q)=%v,%v", tc.in, got, err)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{{0, "0 B"}, {1023, "1023 B"}, {1536, "1.5 KB"}, {5 << 20, "5.0 MB"}, {3 << 30, "3.0 GB"}}
	for _, tc := range tests {
		if got := FormatBytes(tc.in); got != tc.want {
			t.Errorf("FormatBytes(%d)=%q want %q", tc.in, got, tc.want)
		}
	}
}
