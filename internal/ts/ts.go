package ts

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

type Info struct {
	Unix     int64  `json:"unix"`
	UnixMs   int64  `json:"unixMs"`
	ISO      string `json:"iso"`
	UTC      string `json:"utc"`
	Local    string `json:"local"`
	Zone     string `json:"zone"`
	Weekday  string `json:"weekday"`
	Relative string `json:"relative"`
	Detected string `json:"detected"`
}

var digits = regexp.MustCompile(`^\d{1,19}$`)

// Parse nhận diện value và trả thời điểm + tên dạng đã nhận ra.
func Parse(value string, loc *time.Location, now time.Time) (time.Time, string, error) {
	if value == "" || value == "now" {
		return now, "now", nil
	}
	if digits.MatchString(value) {
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil { // 19 chữ số có thể tràn int64
			return time.Time{}, "", fmt.Errorf("số quá lớn: %s", value)
		}
		switch l := len(value); {
		case l <= 11:
			return time.Unix(n, 0), "seconds", nil
		case l <= 14:
			return time.UnixMilli(n), "milliseconds", nil
		case l <= 17:
			return time.UnixMicro(n), "microseconds", nil
		default:
			return time.Unix(0, n), "nanoseconds", nil
		}
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, "rfc3339", nil
	}
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", value, loc); err == nil {
		return t, "datetime", nil
	}
	if t, err := time.ParseInLocation("2006-01-02", value, loc); err == nil {
		return t, "date", nil
	}
	return time.Time{}, "", fmt.Errorf("không nhận diện được %q", value)
}

var weekdays = [7]string{"Chủ Nhật", "Thứ Hai", "Thứ Ba", "Thứ Tư", "Thứ Năm", "Thứ Sáu", "Thứ Bảy"}

func Describe(t time.Time, loc *time.Location, now time.Time) Info {
	lt := t.In(loc)
	name, _ := lt.Zone()
	if loc.String() != "Local" {
		name = loc.String()
	}
	return Info{
		Unix:     t.Unix(),
		UnixMs:   t.UnixMilli(),
		ISO:      lt.Format(time.RFC3339),
		UTC:      t.UTC().Format(time.RFC3339),
		Local:    lt.Format("2006-01-02 15:04:05"),
		Zone:     fmt.Sprintf("%s (%s)", name, lt.Format("-07:00")),
		Weekday:  weekdays[lt.Weekday()],
		Relative: Relative(t, now),
	}
}

// Relative: "3 giờ trước" / "sau 2 ngày" / "vừa xong".
func Relative(t, now time.Time) string {
	d := now.Sub(t)
	future := d < 0
	if future {
		d = -d
	}
	if d < 5*time.Second {
		return "vừa xong"
	}
	units := []struct {
		d    time.Duration
		name string
	}{
		{365 * 24 * time.Hour, "năm"}, {30 * 24 * time.Hour, "tháng"}, {24 * time.Hour, "ngày"},
		{time.Hour, "giờ"}, {time.Minute, "phút"}, {time.Second, "giây"},
	}
	for _, u := range units {
		if d >= u.d {
			n := int64(d / u.d)
			if future {
				return fmt.Sprintf("sau %d %s", n, u.name)
			}
			return fmt.Sprintf("%d %s trước", n, u.name)
		}
	}
	return "vừa xong"
}
