package env

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestParseLines(t *testing.T) {
	tests := []struct{ in, key, value string }{
		{"APP_NAME=Laravel", "APP_NAME", "Laravel"},
		{`APP_NAME="My App"`, "APP_NAME", "My App"},
		{`DB_PASSWORD='p@ss#1'`, "DB_PASSWORD", "p@ss#1"},
		{"MAIL_FROM=a@b.com # comment", "MAIL_FROM", "a@b.com"},
		{"export FOO=bar", "FOO", "bar"},
		{"EMPTY=", "EMPTY", ""},
		{"URL=${APP_URL}/api", "URL", "${APP_URL}/api"},
		{"# comment", "", ""},
		{"", "", ""},
		{`KEY="dòng1\ndòng2"`, "KEY", "dòng1\ndòng2"},
		{"HASH=a#b", "HASH", "a#b"},
		{`Q="say \"hi\" # not comment" # real`, "Q", `say "hi" # not comment`},
		{"A.B=1", "A.B", "1"},
	}
	for _, tc := range tests {
		f, err := Parse(strings.NewReader(tc.in))
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if tc.in == "" {
			continue
		}
		if l := f.Lines[0]; l.Key != tc.key || l.Value != tc.value {
			t.Errorf("%q → key %q value %q", tc.in, l.Key, l.Value)
		}
	}
}

func TestMultilineAndErrors(t *testing.T) {
	f, err := Parse(strings.NewReader("A=1\nPEM=\"-----BEGIN\nabc\n-----END\"\nB=2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := f.Get("PEM"); v != "-----BEGIN\nabc\n-----END" {
		t.Errorf("PEM = %q", v)
	}
	if f.LineOf("B") != 5 {
		t.Errorf("LineOf(B) = %d", f.LineOf("B"))
	}
	for _, bad := range []string{"NOEQUALS", "1BAD=x", "A B=c", `X="unclosed`, "Y='unclosed"} {
		_, err := Parse(strings.NewReader(bad))
		if pe, ok := err.(*ParseError); !ok || pe.Line != 1 {
			t.Errorf("%q: cần ParseError dòng 1, got %v", bad, err)
		}
	}
}

func TestRoundTripAndSet(t *testing.T) {
	src := "# c\r\nA=1\r\n\r\nB=\"x y\"\r\nC=3"
	f, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil || buf.String() != src {
		t.Fatalf("round-trip lệch: %q", buf.String())
	}
	f.Set("A", "two words")
	f.Set("NEW", "v")
	buf.Reset()
	if _, err := f.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	want := "# c\r\nA=\"two words\"\r\n\r\nB=\"x y\"\r\nC=3\r\nNEW=v\r\n"
	if buf.String() != want {
		t.Fatalf("got %q want %q", buf.String(), want)
	}
	if v, _ := f.Get("A"); v != "two words" {
		t.Errorf("Get sau Set = %q", v)
	}
}

func TestDuplicatesKeysGet(t *testing.T) {
	f, _ := Parse(strings.NewReader("A=1\nB=2\nA=3\n"))
	if v, _ := f.Get("A"); v != "3" {
		t.Errorf("trùng key phải lấy giá trị cuối, got %q", v)
	}
	if d := f.Duplicates(); len(d) != 1 || d[0] != "A" {
		t.Errorf("duplicates = %v", d)
	}
	if k := f.Keys(); strings.Join(k, ",") != "A,B" {
		t.Errorf("keys = %v", k)
	}
}

func TestCompareWithFixture(t *testing.T) {
	ex, err := ParseFile("testdata/laravel11.env.example")
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/laravel11.env.example")
	if err != nil {
		t.Fatal(err)
	}
	// .env: bỏ REDIS_HOST, MAIL_MAILER; APP_KEY rỗng ở example nên không tính; thêm OLD_API_KEY; DB_DATABASE rỗng.
	s := strings.NewReplacer("REDIS_HOST=127.0.0.1\n", "", "MAIL_MAILER=log\n", "", "DB_DATABASE=laravel", "DB_DATABASE=").Replace(string(b)) + "OLD_API_KEY=x\n"
	envf, err := Parse(strings.NewReader(s))
	if err != nil {
		t.Fatal(err)
	}
	d := Compare(envf, ex)
	if strings.Join(d.Missing, ",") != "MAIL_MAILER,REDIS_HOST" && strings.Join(d.Missing, ",") != "REDIS_HOST,MAIL_MAILER" {
		t.Errorf("missing = %v", d.Missing)
	}
	if len(d.Extra) != 1 || d.Extra[0] != "OLD_API_KEY" {
		t.Errorf("extra = %v", d.Extra)
	}
	if len(d.Empty) != 1 || d.Empty[0] != "DB_DATABASE" {
		t.Errorf("empty = %v", d.Empty)
	}
	same := Compare(ex, ex)
	if same.Missing == nil || same.Extra == nil || same.Empty == nil || len(same.Missing)+len(same.Extra)+len(same.Empty) != 0 {
		t.Errorf("khớp nhau phải ra 3 slice rỗng không nil: %+v", same)
	}
}

func TestSecretAndMask(t *testing.T) {
	for _, k := range []string{"APP_KEY", "db_password", "GITHUB_TOKEN", "MAIL_PASS"} {
		if !IsSecretKey(k) {
			t.Errorf("%s phải là bí mật", k)
		}
	}
	if IsSecretKey("DB_HOST") {
		t.Error("DB_HOST không phải bí mật")
	}
	if Mask("short") != "***" || Mask("abcdefghij") != "abc***" {
		t.Error("Mask sai")
	}
}
