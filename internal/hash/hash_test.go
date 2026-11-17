package hash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReaderVectors(t *testing.T) {
	tests := []struct {
		name, in, algo, want string
	}{
		{"md5", "abc", "md5", "900150983cd24fb0d6963f7d28e17f72"},
		{"sha1", "abc", "sha1", "a9993e364706816aba3e25717850c26c9cd0d89d"},
		{"sha256", "abc", "sha256", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
		{"crc32", "abc", "crc32", "352441c2"},
		{"sha256 empty", "", "sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"crc32 empty", "", "crc32", "00000000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, n, err := Reader(strings.NewReader(tc.in), []string{tc.algo})
			if err != nil || got[tc.algo] != tc.want || n != int64(len(tc.in)) {
				t.Fatalf("got %v n=%d err=%v", got, n, err)
			}
		})
	}
}

func TestAllAtOnceAndFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, n, err := File(p, Algos)
	if err != nil || n != 3 || len(got) != 5 || got["md5"] != "900150983cd24fb0d6963f7d28e17f72" {
		t.Fatalf("got %v %d %v", got, n, err)
	}
	if _, _, err := File(filepath.Join(t.TempDir(), "none"), Algos); err == nil {
		t.Fatal("cần lỗi file không tồn tại")
	}
	if _, _, err := Reader(strings.NewReader("x"), []string{"zzz"}); err == nil {
		t.Fatal("cần lỗi algo lạ")
	}
}
