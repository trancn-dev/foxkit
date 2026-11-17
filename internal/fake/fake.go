package fake

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"foxkit/internal/pass"
	"foxkit/internal/textcase"
)

type weighted struct {
	V string
	W int
}

type Field struct {
	Kind string
	Arg  string
}

// Key là tên cột: kind (kèm tham số nếu cần để phân biệt hai cột cùng loại thì cmd thêm hậu tố).
func (f Field) Key() string { return f.Kind }

// Kinds: các field hỗ trợ và mô tả ngắn (dùng cho --list).
var Kinds = []struct{ Name, Desc string }{
	{"name", "Họ tên đầy đủ"}, {"first", "Tên"}, {"last", "Họ"}, {"email", "Email (domain ví dụ example.com/org, test.vn)"},
	{"phone", "Số điện thoại VN (phone:intl cho +84)"}, {"address", "Địa chỉ"}, {"city", "Tỉnh/thành"}, {"company", "Tên công ty"},
	{"username", "Tên đăng nhập"}, {"uuid", "UUID v4 (lặp lại được theo seed)"}, {"date", "Ngày, vd date:2020-01-01..2026-12-31"},
	{"datetime", "Ngày giờ ISO"}, {"int", "Số nguyên, vd int:1-100"}, {"bool", "true/false"}, {"text", "Đoạn chữ, vd text:20 (số từ)"},
	{"url", "URL ví dụ"}, {"ip", "IPv4 riêng tư"}, {"password", "Mật khẩu 16 ký tự (crypto/rand, không theo seed)"}, {"pick", "Chọn ngẫu nhiên, vd pick:a|b|c"},
}

type Generator struct {
	rng    *rand.Rand
	locale string
	now    time.Time
	row    *person // người đang sinh trong hàng hiện tại (để email/username khớp name)
}

type person struct {
	first, last, full string
	male              bool
}

// New: seed cố định → kết quả lặp lại được (dữ liệu giả, không phải bí mật nên không dùng crypto/rand).
func New(seed int64, locale string) *Generator {
	return &Generator{rng: rand.New(rand.NewPCG(uint64(seed), uint64(seed))), locale: locale, now: time.Now().Truncate(24 * time.Hour)}
}

func ParseField(s string) (Field, error) {
	kind, arg, _ := strings.Cut(strings.TrimSpace(s), ":")
	kind = strings.ToLower(kind)
	for _, k := range Kinds {
		if k.Name == kind {
			f := Field{Kind: kind, Arg: arg}
			if _, err := New(1, "vi").Value(f); err != nil { // kiểm tham số bằng cách thử sinh một giá trị
				return Field{}, err
			}
			return f, nil
		}
	}
	return Field{}, fmt.Errorf("field không hợp lệ: %q", s)
}

func (g *Generator) pick(list []string) string { return list[g.rng.IntN(len(list))] }

func (g *Generator) weightedPick(list []weighted) string {
	total := 0
	for _, w := range list {
		total += w.W
	}
	n := g.rng.IntN(total)
	for _, w := range list {
		if n < w.W {
			return w.V
		}
		n -= w.W
	}
	return list[len(list)-1].V
}

func (g *Generator) newPerson() *person {
	p := &person{male: g.rng.IntN(2) == 0}
	if g.locale == "en" {
		if p.male {
			p.first = g.pick(enFirstMale)
		} else {
			p.first = g.pick(enFirstFemale)
		}
		p.last = g.pick(enSurnames)
		p.full = p.first + " " + p.last
		return p
	}
	p.last = g.weightedPick(viSurnames)
	var middle string
	if p.male {
		middle, p.first = g.pick(viMiddleMale), g.pick(viFirstMale)
	} else {
		middle, p.first = g.pick(viMiddleFemale), g.pick(viFirstFemale)
	}
	p.full = p.last + " " + middle + " " + p.first
	return p
}

func (g *Generator) person() *person {
	if g.row == nil {
		g.row = g.newPerson()
	}
	return g.row
}

var emailDomains = []string{"example.com", "example.org", "test.vn"}

func slug(s string) string {
	out, _ := textcase.Convert("slug", s)
	return strings.ReplaceAll(out, "-", ".")
}

func (g *Generator) intRange(arg string, lo, hi int) (int, error) {
	if arg != "" {
		a, b, ok := strings.Cut(arg, "-")
		l, err1 := strconv.Atoi(a)
		h, err2 := strconv.Atoi(b)
		if !ok || err1 != nil || err2 != nil || h < l {
			return 0, fmt.Errorf("int cần dạng int:min-max, nhận %q", arg)
		}
		lo, hi = l, h
	}
	return lo + g.rng.IntN(hi-lo+1), nil
}

func (g *Generator) dateIn(arg string, withTime bool) (string, error) {
	start, end := g.now.AddDate(-5, 0, 0), g.now
	if arg != "" {
		a, b, ok := strings.Cut(arg, "..")
		s, err1 := time.Parse("2006-01-02", a)
		e, err2 := time.Parse("2006-01-02", b)
		if !ok || err1 != nil || err2 != nil || e.Before(s) {
			return "", fmt.Errorf("date cần dạng date:2020-01-01..2026-12-31, nhận %q", arg)
		}
		start, end = s, e
	}
	span := int64(end.Sub(start) / time.Second)
	t := start.Add(time.Duration(g.rng.Int64N(span+1)) * time.Second)
	if withTime {
		return t.UTC().Format("2006-01-02T15:04:05Z"), nil
	}
	return t.Format("2006-01-02"), nil
}

// Value sinh một giá trị cho field.
func (g *Generator) Value(f Field) (any, error) {
	en := g.locale == "en"
	switch f.Kind {
	case "name":
		return g.person().full, nil
	case "first":
		return g.person().first, nil
	case "last":
		return g.person().last, nil
	case "email":
		p := g.person()
		return fmt.Sprintf("%s.%s%d@%s", slug(p.first), slug(p.last), g.rng.IntN(99)+1, g.pick(emailDomains)), nil
	case "username":
		p := g.person()
		return fmt.Sprintf("%s%s%d", strings.ReplaceAll(slug(p.first), ".", ""), strings.ReplaceAll(slug(p.last), ".", ""), g.rng.IntN(99)+1), nil
	case "phone":
		prefix := g.pick([]string{"03", "05", "07", "08", "09"})
		num := prefix + fmt.Sprintf("%d", g.rng.IntN(10)) + fmt.Sprintf("%07d", g.rng.IntN(10000000))
		if f.Arg == "intl" {
			return "+84" + num[1:], nil
		}
		return num[:4] + " " + num[4:7] + " " + num[7:], nil
	case "address":
		if en {
			return fmt.Sprintf("%d %s, %s", g.rng.IntN(999)+1, g.pick(enStreets), g.pick(enCities)), nil
		}
		pl := viPlaces[g.rng.IntN(len(viPlaces))]
		return fmt.Sprintf("%d %s, %s, %s", g.rng.IntN(300)+1, g.pick(viStreets), g.pick(pl.Districts), pl.City), nil
	case "city":
		if en {
			return g.pick(enCities), nil
		}
		return viPlaces[g.rng.IntN(len(viPlaces))].City, nil
	case "company":
		if en {
			return g.pick(enCompanies), nil
		}
		return g.pick(viCompanyKinds) + " " + g.pick(viCompanyWords), nil
	case "uuid":
		var b [16]byte
		for i := range b {
			b[i] = byte(g.rng.UintN(256))
		}
		b[6], b[8] = (b[6]&0x0f)|0x40, (b[8]&0x3f)|0x80
		return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
	case "date":
		return g.dateIn(f.Arg, false)
	case "datetime":
		return g.dateIn(f.Arg, true)
	case "int":
		return g.intRange(f.Arg, 0, 1000000)
	case "bool":
		return g.rng.IntN(2) == 0, nil
	case "text":
		n := 12
		if f.Arg != "" {
			var err error
			if n, err = strconv.Atoi(f.Arg); err != nil || n < 1 || n > 1000 {
				return nil, fmt.Errorf("text cần dạng text:số-từ (1-1000), nhận %q", f.Arg)
			}
		}
		words := viWords
		if en {
			words = enWords
		}
		parts := make([]string, n)
		for i := range parts {
			parts[i] = g.pick(words)
		}
		s := strings.Join(parts, " ")
		return strings.ToUpper(s[:1]) + s[1:] + ".", nil
	case "url":
		p := g.person()
		return fmt.Sprintf("https://%s.example.com/%s", strings.ReplaceAll(slug(p.last), ".", ""), strings.ReplaceAll(slug(p.first), ".", "")), nil
	case "ip":
		return fmt.Sprintf("192.168.%d.%d", g.rng.IntN(256), g.rng.IntN(254)+1), nil
	case "password":
		return pass.Generate(pass.Options{Length: 16, Charset: "alnum"})
	case "pick":
		opts := strings.Split(f.Arg, "|")
		if f.Arg == "" || len(opts) < 1 {
			return nil, fmt.Errorf("pick cần dạng pick:a|b|c")
		}
		return g.pick(opts), nil
	}
	return nil, fmt.Errorf("field không hợp lệ: %q", f.Kind)
}

// Row giữ thứ tự cột (map của Go không có thứ tự) và ra JSON đúng thứ tự.
type Row struct {
	Keys []string
	Vals []any
}

func (r Row) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, k := range r.Keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		vb, err := json.Marshal(r.Vals[i])
		if err != nil {
			return nil, err
		}
		b.Write(kb)
		b.WriteByte(':')
		b.Write(vb)
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}

// Keys đặt tên cột; hai field cùng loại thì thêm hậu tố _2, _3...
func Keys(fields []Field) []string {
	seen := map[string]int{}
	keys := make([]string, len(fields))
	for i, f := range fields {
		seen[f.Kind]++
		keys[i] = f.Kind
		if n := seen[f.Kind]; n > 1 {
			keys[i] = fmt.Sprintf("%s_%d", f.Kind, n)
		}
	}
	return keys
}

// Rows sinh n hàng; field email/username/name... trong cùng hàng dùng lại cùng một người.
func (g *Generator) Rows(fields []Field, n int) ([]Row, error) {
	keys := Keys(fields)
	rows := make([]Row, 0, n)
	for i := 0; i < n; i++ {
		g.row = nil
		vals := make([]any, len(fields))
		for j, f := range fields {
			v, err := g.Value(f)
			if err != nil {
				return nil, err
			}
			vals[j] = v
		}
		rows = append(rows, Row{Keys: keys, Vals: vals})
	}
	return rows, nil
}
