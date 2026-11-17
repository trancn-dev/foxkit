package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strings"

	"foxkit/internal/fsutil"
	"foxkit/internal/hash"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

type hashItem struct {
	Source string            `json:"source"`
	Bytes  int64             `json:"bytes"`
	Hashes map[string]string `json:"hashes,omitempty"`
	Match  *bool             `json:"match,omitempty"`
	Error  string            `json:"error,omitempty"`
}

func init() {
	c := &cobra.Command{
		Use:   "hash [file...]",
		Short: "Tính md5/sha1/sha256/sha512/crc32 của file hoặc chuỗi",
		RunE: func(c *cobra.Command, files []string) error {
			algo, _ := c.Flags().GetString("algo")
			all, _ := c.Flags().GetBool("all")
			text, _ := c.Flags().GetString("text")
			check, _ := c.Flags().GetString("check")
			if !slices.Contains(hash.Algos, algo) {
				return output.Usage("HASH_ALGO_INVALID", "Thuật toán không hợp lệ: "+algo, "Chọn: md5, sha1, sha256, sha512, crc32")
			}
			hasText := c.Flags().Changed("text")
			if !hasText && len(files) == 0 {
				return output.Usage("HASH_NO_INPUT", "Thiếu file hoặc --text", "Ví dụ: foxkit hash setup.exe")
			}
			if hasText && len(files) > 0 {
				return output.Usage("HASH_ARGS_CONFLICT", "Không dùng --text cùng với file", "Chỉ chọn một nguồn")
			}
			sources := files
			if hasText {
				sources = []string{"text"}
			}
			if check != "" && len(sources) != 1 {
				return output.Usage("HASH_ARGS_CONFLICT", "--check chỉ dùng được với đúng một nguồn", "Chạy riêng từng file")
			}
			algos := []string{algo}
			if all {
				algos = hash.Algos
			}

			items := make([]hashItem, 0, len(sources))
			failed, mismatch := false, false
			for _, s := range sources {
				it := hashItem{Source: s}
				var hs map[string]string
				var err error
				switch {
				case hasText:
					hs, it.Bytes, err = hash.Reader(strings.NewReader(text), algos)
				case s == "-":
					it.Source = "stdin"
					hs, it.Bytes, err = hash.Reader(c.InOrStdin(), algos)
				default:
					hs, it.Bytes, err = hash.File(s, algos)
				}
				if err != nil {
					failed = true
					it.Error = err.Error()
					if errors.Is(err, fs.ErrNotExist) {
						it.Error = "Không tìm thấy file " + s
					}
				} else {
					it.Hashes = hs
					if check != "" {
						m := strings.EqualFold(hs[algo], strings.TrimSpace(check))
						it.Match = &m
						mismatch = !m
					}
				}
				items = append(items, it)
			}

			err := printer(c).Success(map[string]any{"items": items}, func(w io.Writer) error {
				for _, it := range items {
					switch {
					case it.Error != "":
						fmt.Fprintf(c.ErrOrStderr(), "Lỗi: %s\n", it.Error)
					case all:
						fmt.Fprintf(w, "%s (%s)\n", it.Source, fsutil.FormatBytes(it.Bytes))
						for _, a := range algos {
							fmt.Fprintf(w, "  %-7s%s\n", a, it.Hashes[a])
						}
					default:
						fmt.Fprintf(w, "%s  %s\n", it.Hashes[algo], it.Source)
					}
					if it.Match != nil {
						fmt.Fprintf(w, "khớp: %v\n", *it.Match)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
			if mismatch {
				return output.Fail("HASH_MISMATCH", "Hash không khớp với giá trị --check", "File có thể tải lỗi hoặc bị sửa, hãy tải lại")
			}
			if failed {
				return output.Fail("FILE_NOT_FOUND", "Có file không đọc được", "Kiểm tra lại đường dẫn")
			}
			return nil
		},
	}
	c.Flags().String("algo", "sha256", "md5|sha1|sha256|sha512|crc32")
	c.Flags().Bool("all", false, "Tính cả 5 thuật toán")
	c.Flags().String("text", "", "Hash chuỗi thay vì file")
	c.Flags().String("check", "", "Hash cần so sánh")
	rootCmd.AddCommand(c)
}
