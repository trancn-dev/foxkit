package svc

import (
	"runtime"
	"strings"
	"testing"
)

func TestNames(t *testing.T) {
	for st, want := range map[uint32]string{1: "stopped", 2: "starting", 3: "stopping", 4: "running", 7: "paused", 99: "unknown"} {
		if StateName(st) != want {
			t.Errorf("StateName(%d) != %s", st, want)
		}
	}
	if StartTypeName(2, false) != "auto" || StartTypeName(2, true) != "auto-delayed" || StartTypeName(3, false) != "manual" || StartTypeName(4, false) != "disabled" {
		t.Error("StartTypeName sai")
	}
}

func TestDevFilterAndMatches(t *testing.T) {
	for _, tc := range []struct {
		name, display string
		want          bool
	}{
		{"MySQL80", "MySQL80", true}, {"Redis", "Redis", true}, {"com.docker.service", "Docker Desktop Service", true},
		{"postgresql-x64-16", "postgresql", true}, {"Dnscache", "DNS Client", false}, {"W3SVC", "World Wide Web Publishing (nginx proxy)", true},
	} {
		if IsDev(tc.name, tc.display) != tc.want {
			t.Errorf("IsDev(%q,%q) != %v", tc.name, tc.display, tc.want)
		}
	}
	if !Matches("Dnscache", "DNS Client", "client") || Matches("Dnscache", "DNS Client", "mysql") {
		t.Error("Matches sai")
	}
}

func TestSuggest(t *testing.T) {
	got := Suggest([]string{"MySQL80", "MySQL57", "Redis", "mysqld", "MySQLx"}, "mysql")
	if len(got) != 3 || got[0] != "MySQL80" {
		t.Fatalf("%v", got)
	}
	if len(Suggest([]string{"a"}, "zzz")) != 0 {
		t.Error("không có gợi ý phải rỗng")
	}
}

func TestListWithoutAdmin(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("chỉ Windows")
	}
	all, err := List("", true)
	if err != nil {
		t.Fatalf("liệt kê phải chạy được khi không phải admin: %v", err)
	}
	found := false
	for _, s := range all {
		if strings.EqualFold(s.Name, "Dnscache") || strings.EqualFold(s.Name, "EventLog") {
			found = true
			if s.State == "" || s.StartType == "" {
				t.Errorf("%+v", s)
			}
		}
	}
	if !found {
		t.Fatalf("thiếu Dnscache/EventLog trong %d service", len(all))
	}
	if _, err := Start("khong-co-service-nay-xyz", 0); err == nil || !strings.Contains(err.Error(), "không tìm thấy") {
		t.Errorf("service lạ phải báo không tìm thấy: %v", err)
	}
}
