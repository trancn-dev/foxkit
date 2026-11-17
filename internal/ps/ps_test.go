package ps

import (
	"os"
	"testing"
)

func load(t *testing.T, name string) []Proc {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	ps, err := Parse(string(b))
	if err != nil {
		t.Fatal(err)
	}
	return ps
}

func TestParseBothLocales(t *testing.T) {
	en := load(t, "tasklist_en.csv")
	if len(en) != 6 || en[2].PID != 9812 || en[2].MemBytes != 412345*1024 || en[2].Session != "Console" {
		t.Fatalf("en: %+v", en[2])
	}
	de := load(t, "tasklist_de.csv")
	if de[1].MemBytes != 412345*1024 || de[2].MemBytes != 1024*1024 {
		t.Fatalf("de (dấu chấm ngăn nghìn): %+v", de)
	}
}

func TestFilterAndGroup(t *testing.T) {
	ps := load(t, "tasklist_en.csv")
	if f := Filter(ps, "NODE"); len(f) != 3 {
		t.Fatalf("filter: %+v", f)
	}
	gs := GroupByName(Filter(ps, "node"))
	if len(gs) != 1 || gs[0].Count != 3 || gs[0].MemBytes != (412345+100000+5000)*1024 || len(gs[0].PIDs) != 3 {
		t.Fatalf("group: %+v", gs)
	}
	all := GroupByName(ps)
	SortGroups(all, "mem")
	if all[0].Name != "node.exe" && all[0].Name != "chrome.exe" {
		t.Fatalf("%+v", all)
	}
	if all[0].MemBytes < all[1].MemBytes {
		t.Fatal("phải giảm dần theo mem")
	}
}

func TestSortStable(t *testing.T) {
	ps := load(t, "tasklist_en.csv")
	Sort(ps, "mem")
	if ps[0].Name != "chrome.exe" {
		t.Fatalf("mem: %+v", ps[0])
	}
	Sort(ps, "pid")
	if ps[0].PID != 0 || ps[1].PID != 4 {
		t.Fatalf("pid: %+v", ps[:2])
	}
	Sort(ps, "name")
	if ps[0].Name != "chrome.exe" {
		t.Fatalf("name: %+v", ps[0])
	}
}

func TestEmptyFilterIsNotNil(t *testing.T) {
	if f := Filter(load(t, "tasklist_en.csv"), "zzz"); f == nil || len(f) != 0 {
		t.Fatal("phải là slice rỗng không nil")
	}
}
