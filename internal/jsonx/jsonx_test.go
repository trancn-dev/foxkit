package jsonx

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestPrettyKeepsOrderAndNumbers(t *testing.T) {
	in := []byte(`{"b":1,"a":2,"id":12345678901234567890,"f":1.0}`)
	out, err := Pretty(in, "  ")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Index(s, `"b"`) > strings.Index(s, `"a"`) || !strings.Contains(s, "12345678901234567890") || !strings.Contains(s, "1.0") {
		t.Fatalf("mất thứ tự/số: %s", s)
	}
	m, _ := Minify(out)
	if string(m) != string(in) {
		t.Fatalf("minify: %s", m)
	}
}

func TestSortKeysKeepsBigNumbers(t *testing.T) {
	out, err := SortKeys([]byte(`{"b":12345678901234567890,"a":"<x>"}`), "")
	if err != nil || string(out) != `{"a":"<x>","b":12345678901234567890}` {
		t.Fatalf("%s %v", out, err)
	}
}

func TestValidatePosition(t *testing.T) {
	e := Validate([]byte(`{"a":1,}`))
	if e == nil || e.Line != 1 || e.Col != 8 {
		t.Fatalf("%+v", e)
	}
	e = Validate([]byte("{\n  \"a\": 1,\n  \"b\": ]\n}"))
	if e == nil || e.Line != 3 || !strings.Contains(e.Context, "^") {
		t.Fatalf("%+v", e)
	}
	if Validate([]byte(`{"ok":true}`)) != nil {
		t.Error("hợp lệ phải nil")
	}
	if Validate([]byte("\xef\xbb\xbf{}")) != nil {
		t.Error("BOM phải được bỏ")
	}
	if e := Validate([]byte(`{"a":`)); e == nil {
		t.Error("thiếu giá trị phải lỗi")
	}
}

func TestGet(t *testing.T) {
	doc := []byte(`{"a":{"b":[{"v":10},{"v":20}]},"items":[{"id":1},{"id":2},{"x":1}],"x.y":"dot","s":"text"}`)
	tests := []struct {
		path string
		want string
	}{
		{"a.b.0.v", `10`},
		{"items.#", `3`},
		{"items.*.id", `[1,2]`},
		{"items.-1.x", `1`},
		{`["x.y"]`, `"dot"`},
		{"s", `"text"`},
		{"a.b.#", `2`},
	}
	for _, tc := range tests {
		v, err := Get(doc, tc.path)
		if err != nil {
			t.Errorf("%s: %v", tc.path, err)
			continue
		}
		b, _ := json.Marshal(v)
		if string(b) != tc.want {
			t.Errorf("%s = %s, cần %s", tc.path, b, tc.want)
		}
	}
}

func TestGetErrors(t *testing.T) {
	doc := []byte(`{"data":{"items":[]}}`)
	_, err := Get(doc, "data.items.0")
	var nf *NotFound
	if !errors.As(err, &nf) || nf.Walked != "data.items" || nf.Missing != "0" {
		t.Fatalf("%v", err)
	}
	var pe *PathError
	for _, bad := range []string{"", `["a`, `[a]`, `["a"x`} {
		if _, err := Get(doc, bad); !errors.As(err, &pe) {
			t.Errorf("%q phải là PathError, got %v", bad, err)
		}
	}
}

func TestParsePath(t *testing.T) {
	toks, err := ParsePath(`require.laravel/framework`)
	if err != nil || len(toks) != 2 || toks[1] != "laravel/framework" {
		t.Fatalf("%v %v", toks, err)
	}
	toks, _ = ParsePath(`a["b.c"].d`)
	if strings.Join(toks, "|") != "a|b.c|d" {
		t.Fatalf("%v", toks)
	}
}
