package dns

import (
	"context"
	"testing"
)

func TestNormalizeHost(t *testing.T) {
	for in, want := range map[string]string{
		"https://a.com/x": "a.com", "a.com:8080": "a.com", "http://b.org:3000/p?q=1": "b.org", "  c.net ": "c.net", "d.io/path": "d.io", "plain": "plain",
	} {
		if got := NormalizeHost(in); got != want {
			t.Errorf("NormalizeHost(%q) = %q, cần %q", in, got, want)
		}
	}
}

func TestLookupLocalhostNeedsNoNetwork(t *testing.T) {
	r := Lookup(context.Background(), "localhost", []string{"A", "MX"}, "")
	if len(r.Records.A) == 0 {
		t.Skip("máy này không phân giải localhost qua resolver:", r.Errors)
	}
	if r.Records.A[0] != "127.0.0.1" || r.Server != "system" || r.Errors == nil {
		t.Fatalf("%+v", r)
	}
	if _, ok := r.Errors["MX"]; !ok {
		t.Error("localhost không có MX, phải nằm trong Errors")
	}
}

func TestLookupRealDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("cần mạng")
	}
	r := Lookup(context.Background(), "example.com", []string{"A"}, "")
	if len(r.Records.A) == 0 {
		t.Logf("không tra được example.com (có thể offline): %v", r.Errors)
	}
}
