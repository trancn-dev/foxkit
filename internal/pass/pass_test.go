package pass

import (
	"strings"
	"testing"
)

func TestGenerateCharsets(t *testing.T) {
	tests := []struct {
		charset string
		noAmb   bool
	}{{"all", false}, {"alnum", false}, {"alpha", false}, {"num", false}, {"hex", false}, {"all", true}, {"alnum", true}}
	for _, tc := range tests {
		alpha, err := Alphabet(tc.charset, tc.noAmb)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 200; i++ {
			s, err := Generate(Options{Length: 32, Charset: tc.charset, NoAmbiguous: tc.noAmb})
			if err != nil || len(s) != 32 {
				t.Fatalf("%s: len %d err %v", tc.charset, len(s), err)
			}
			for _, r := range s {
				if !strings.ContainsRune(alpha, r) {
					t.Fatalf("%s: ký tự %q ngoài alphabet", tc.charset, r)
				}
				if tc.noAmb && strings.ContainsRune(ambiguous, r) {
					t.Fatalf("còn ký tự dễ nhầm %q", r)
				}
			}
		}
	}
}

func TestAllHasFourClasses(t *testing.T) {
	for i := 0; i < 1000; i++ {
		s, err := Generate(Options{Length: 4, Charset: "all"})
		if err != nil || !hasAllClasses(s) {
			t.Fatalf("thiếu loại ký tự: %q %v", s, err)
		}
	}
}

func TestBadCharsetAndEntropy(t *testing.T) {
	if _, err := Generate(Options{Length: 8, Charset: "zzz"}); err == nil {
		t.Fatal("cần lỗi")
	}
	if got := EntropyBits(24, 64); got != 144 {
		t.Fatalf("got %v", got)
	}
}
