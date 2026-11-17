package hash

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	stdhash "hash"
	"hash/crc32"
	"io"
	"os"
)

// Algos là danh sách thuật toán hỗ trợ, theo thứ tự hiển thị của --all.
var Algos = []string{"md5", "sha1", "sha256", "sha512", "crc32"}

func New(algo string) (stdhash.Hash, error) {
	switch algo {
	case "md5":
		return md5.New(), nil
	case "sha1":
		return sha1.New(), nil
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	case "crc32":
		return crc32.NewIEEE(), nil
	}
	return nil, fmt.Errorf("thuật toán không hỗ trợ: %q", algo)
}

// Reader đọc r một lần, tính mọi algo cùng lúc; trả hex theo algo và số byte đã đọc.
func Reader(r io.Reader, algos []string) (map[string]string, int64, error) {
	hs := make([]stdhash.Hash, len(algos))
	ws := make([]io.Writer, len(algos))
	for i, a := range algos {
		h, err := New(a)
		if err != nil {
			return nil, 0, err
		}
		hs[i], ws[i] = h, h
	}
	n, err := io.Copy(io.MultiWriter(ws...), r)
	if err != nil {
		return nil, n, err
	}
	out := make(map[string]string, len(algos))
	for i, a := range algos {
		out[a] = hex.EncodeToString(hs[i].Sum(nil))
	}
	return out, n, nil
}

func File(path string, algos []string) (map[string]string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	return Reader(f, algos)
}
