package csvx

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func read(t *testing.T, s string) *Table {
	t.Helper()
	tb, err := Read(strings.NewReader(s), 0, false)
	if err != nil {
		t.Fatal(err)
	}
	return tb
}

func TestDetectDelim(t *testing.T) {
	tests := []struct {
		in   string
		want rune
	}{
		{"a,b,c\n1,2,3", ','},
		{"a;b;c\n1;2;3", ';'},
		{"\"a;b\",c,d\n\"x;y\",2,3", ','},
		{"a\tb\tc\n1\t2\t3", '\t'},
		{"a|b|c\n1|2|3", '|'},
		{"chỉ một cột", ','},
		{"", ','},
	}
	for _, tc := range tests {
		if got := DetectDelim([]byte(tc.in)); got != tc.want {
			t.Errorf("DetectDelim(%q) = %q, cần %q", tc.in, got, tc.want)
		}
	}
}

func TestReadBOMCRLFMultilineQuotedAndPadding(t *testing.T) {
	tb := read(t, "\xef\xbb\xbfid;name;note\r\n1;Hùng;\"dòng 1\ndòng 2\"\r\n2;Lan\r\n")
	if !tb.HadBOM || tb.Delim != ';' || len(tb.Rows) != 2 {
		t.Fatalf("%+v", tb)
	}
	if tb.Rows[0][2] != "dòng 1\ndòng 2" {
		t.Errorf("ô nhiều dòng trong nháy: %q", tb.Rows[0][2])
	}
	if len(tb.Rows[1]) != 3 || tb.Rows[1][2] != "" || len(tb.Warnings) != 1 {
		t.Errorf("dòng thiếu cột phải được đệm + warning: %+v %v", tb.Rows[1], tb.Warnings)
	}
	nh, _ := Read(strings.NewReader("1,2\n3,4"), ',', true)
	if nh.Header[0] != "c1" || len(nh.Rows) != 2 {
		t.Errorf("--no-header: %+v", nh)
	}
}

func TestEncodingError(t *testing.T) {
	if _, err := Read(strings.NewReader("a,b\n\xe9,2"), 0, false); !errors.Is(err, ErrEncoding) {
		t.Fatalf("%v", err)
	}
}

func TestSelectAndWhere(t *testing.T) {
	tb := read(t, "id,name,age,city\n1,Hùng,17,Hà Nội\n2,Lan,25,Hà Nội\n3,Minh,30,Đà Nẵng\n4,An,40,Hà Nội\n")
	w, err := tb.Where("age>18,city=Hà Nội")
	if err != nil || len(w.Rows) != 2 || w.Rows[0][1] != "Lan" {
		t.Fatalf("%+v %v", w, err)
	}
	if w, _ := tb.Where("name~ân"); len(w.Rows) != 0 {
		t.Errorf("~ là chứa chuỗi con: %+v", w.Rows)
	}
	if w, _ := tb.Where("age!=25"); len(w.Rows) != 3 {
		t.Errorf("!=: %d", len(w.Rows))
	}
	if w, _ := tb.Where("age>=30,age<=40"); len(w.Rows) != 2 {
		t.Errorf(">= <=: %d", len(w.Rows))
	}
	s, err := tb.Select([]string{"2", "NAME", "city"})
	if err != nil || strings.Join(s.Header, ",") != "name,name,city" || s.Rows[0][0] != "Hùng" {
		t.Fatalf("%+v %v", s, err)
	}
	var ce *ColError
	if _, err := tb.Select([]string{"nope"}); !errors.As(err, &ce) || len(ce.Have) != 4 {
		t.Errorf("%v", err)
	}
	var we *WhereError
	if _, err := tb.Where("khong co toan tu"); !errors.As(err, &we) {
		t.Errorf("%v", err)
	}
	if _, err := tb.Where("zzz=1"); !errors.As(err, &ce) {
		t.Errorf("cột lạ phải ColError: %v", err)
	}
}

func TestStats(t *testing.T) {
	tb := read(t, "id,price,when,city\n1,1.5,2026-01-02,Hà Nội\n2,2.5,2026-03-04,\n3,x,2026-02-01,Hà Nội\n")
	st := tb.Stats()
	if st[0].Kind != "int" || st[0].Min != "1" || st[0].Max != "3" || st[0].Unique != "3" {
		t.Errorf("id: %+v", st[0])
	}
	if st[1].Kind == "float" {
		t.Errorf("price có 'x' (<95%% số) phải là text: %+v", st[1])
	}
	if st[2].Kind != "date" || st[2].Min != "2026-01-02" || st[2].Max != "2026-03-04" {
		t.Errorf("when: %+v", st[2])
	}
	if st[3].Empty != 1 || st[3].Unique != "1" || st[3].Kind != "text" {
		t.Errorf("city: %+v", st[3])
	}
}

func TestObjectsKeepOrderAndVietnamese(t *testing.T) {
	tb := read(t, "z,a\n1,Hà Nội\n")
	b, err := json.Marshal(tb.Objects(true))
	if err != nil || string(b) != `[{"z":1,"a":"Hà Nội"}]` {
		t.Fatalf("%s %v", b, err)
	}
	b, _ = json.Marshal(tb.Objects(false))
	if string(b) != `[{"z":"1","a":"Hà Nội"}]` {
		t.Fatalf("%s", b)
	}
	for in, want := range map[string]any{"007": "007", "12": int64(12), "1.5": 1.5, "true": true, "NaN": "NaN", "abc": "abc"} {
		if got := Typed(in); got != want {
			t.Errorf("Typed(%q) = %#v, cần %#v", in, got, want)
		}
	}
}
