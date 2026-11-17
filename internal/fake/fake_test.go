package fake

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"
)

func fields(t *testing.T, specs ...string) []Field {
	t.Helper()
	var out []Field
	for _, s := range specs {
		f, err := ParseField(s)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, f)
	}
	return out
}

func rowsJSON(t *testing.T, seed int64, locale string, specs ...string) string {
	rows, err := New(seed, locale).Rows(fields(t, specs...), 20)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSameSeedSameBytesDifferentSeedDiffers(t *testing.T) {
	a := rowsJSON(t, 1, "vi", "name", "email", "phone", "address", "uuid", "text:5")
	b := rowsJSON(t, 1, "vi", "name", "email", "phone", "address", "uuid", "text:5")
	c := rowsJSON(t, 2, "vi", "name", "email", "phone", "address", "uuid", "text:5")
	if a != b {
		t.Error("cùng seed phải giống hệt")
	}
	if a == c {
		t.Error("khác seed phải khác")
	}
}

func TestPhoneFormat(t *testing.T) {
	re := regexp.MustCompile(`^0(3|5|7|8|9)\d{8}$`)
	g := New(7, "vi")
	for i := 0; i < 1000; i++ {
		v, _ := g.Value(Field{Kind: "phone"})
		if !re.MatchString(strings.ReplaceAll(v.(string), " ", "")) {
			t.Fatalf("phone sai: %v", v)
		}
	}
	v, _ := g.Value(Field{Kind: "phone", Arg: "intl"})
	if !strings.HasPrefix(v.(string), "+84") {
		t.Errorf("intl: %v", v)
	}
}

func TestGenderNamesAndEmail(t *testing.T) {
	g := New(3, "vi")
	for i := 0; i < 500; i++ {
		g.row = nil
		name, _ := g.Value(Field{Kind: "name"})
		email, _ := g.Value(Field{Kind: "email"})
		n := name.(string)
		if strings.Contains(n, " Thị ") && strings.Contains(n, " Văn ") {
			t.Fatalf("lẫn Thị/Văn: %s", n)
		}
		e := email.(string)
		ok := strings.HasSuffix(e, "@example.com") || strings.HasSuffix(e, "@example.org") || strings.HasSuffix(e, "@test.vn")
		for _, r := range e {
			if r > 127 {
				ok = false
			}
		}
		if !ok {
			t.Fatalf("email sai: %s", e)
		}
	}
}

func TestFemaleHasNoVanMaleHasNoThi(t *testing.T) {
	g := New(9, "vi")
	for i := 0; i < 2000; i++ {
		p := g.newPerson()
		if p.male && strings.Contains(p.full, " Thị ") {
			t.Fatalf("nam có Thị: %s", p.full)
		}
		if !p.male && strings.Contains(p.full, " Văn ") {
			t.Fatalf("nữ có Văn: %s", p.full)
		}
	}
}

func TestEmailConsistentWithNameInRow(t *testing.T) {
	rows, _ := New(5, "vi").Rows(fields(t, "name", "email"), 50)
	for _, r := range rows {
		name := r.Vals[0].(string)
		email := r.Vals[1].(string)
		last := strings.Fields(name)[0]
		if !strings.Contains(email, strings.ToLower(slug(last))) {
			t.Fatalf("email %q không khớp họ của %q", email, name)
		}
	}
}

func TestParametrizedFields(t *testing.T) {
	g := New(1, "vi")
	for i := 0; i < 200; i++ {
		v, err := g.Value(Field{Kind: "int", Arg: "1-5"})
		if err != nil || v.(int) < 1 || v.(int) > 5 {
			t.Fatalf("int:1-5 → %v %v", v, err)
		}
		p, _ := g.Value(Field{Kind: "pick", Arg: "a|b"})
		if p != "a" && p != "b" {
			t.Fatalf("pick → %v", p)
		}
		d, err := g.Value(Field{Kind: "date", Arg: "2020-01-01..2020-12-31"})
		if _, perr := time.Parse("2006-01-02", d.(string)); err != nil || perr != nil || !strings.HasPrefix(d.(string), "2020-") {
			t.Fatalf("date → %v %v", d, err)
		}
	}
	for _, bad := range []string{"int:x", "int:5-1", "date:abc", "text:0", "zzz", "pick"} {
		if _, err := ParseField(bad); err == nil {
			t.Errorf("%q phải lỗi", bad)
		}
	}
	if v, _ := g.Value(Field{Kind: "text", Arg: "7"}); len(strings.Fields(v.(string))) != 7 {
		t.Errorf("text:7 → %v", v)
	}
}

func TestKeysAndOrderedJSON(t *testing.T) {
	keys := Keys(fields(t, "int:1-5", "name", "int:1-9"))
	if strings.Join(keys, ",") != "int,name,int_2" {
		t.Fatalf("%v", keys)
	}
	b, _ := json.Marshal(Row{Keys: []string{"z", "a"}, Vals: []any{1, "x"}})
	if string(b) != `{"z":1,"a":"x"}` {
		t.Fatalf("thứ tự cột phải giữ nguyên: %s", b)
	}
}

func TestEnglishLocaleAndSpeed(t *testing.T) {
	v, _ := New(1, "en").Value(Field{Kind: "name"})
	if strings.ContainsAny(v.(string), "ăâđêôơư") {
		t.Errorf("locale en: %v", v)
	}
	start := time.Now()
	if _, err := New(1, "vi").Rows(fields(t, "name", "email", "phone", "address", "date"), 100000); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("100.000 dòng phải dưới 2s, mất %v", time.Since(start))
	}
}
