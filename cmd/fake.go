package cmd

import (
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"
	"time"

	"foxkit/internal/fake"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

func init() {
	c := &cobra.Command{
		Use:   "fake <field>...",
		Short: "Sinh dữ liệu giả tiếng Việt: tên, SĐT, địa chỉ, email...",
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			list, _ := f.GetBool("list")
			count, _ := f.GetInt("count")
			format, _ := f.GetString("format")
			locale, _ := f.GetString("locale")
			bom, _ := f.GetBool("bom")
			p := printer(c)
			if list {
				return p.Success(map[string]any{"fields": fake.Kinds}, func(w io.Writer) error {
					for _, k := range fake.Kinds {
						fmt.Fprintf(w, "%-10s %s\n", k.Name, k.Desc)
					}
					return nil
				})
			}
			if len(args) == 0 {
				return output.Usage("FAKE_NO_FIELD", "Thiếu field cần sinh", "Ví dụ: foxkit fake name email --count 5")
			}
			if count < 1 || count > 100000 {
				return output.Usage("FAKE_COUNT_INVALID", "--count phải từ 1 đến 100000", "Từ 1 đến 100000")
			}
			if !slices.Contains([]string{"table", "json", "csv", "ndjson"}, format) {
				return output.Usage("ARGS_INVALID", "--format phải là table, json, csv hoặc ndjson", "")
			}
			if locale != "vi" && locale != "en" {
				return output.Usage("ARGS_INVALID", "--locale phải là vi hoặc en", "")
			}
			fields := make([]fake.Field, len(args))
			for i, a := range args {
				fd, err := fake.ParseField(a)
				if err != nil {
					return output.Usage("FAKE_FIELD_INVALID", err.Error(), "Xem danh sách: foxkit fake --list")
				}
				fields[i] = fd
			}
			seed := time.Now().UnixNano()
			if f.Changed("seed") {
				seed, _ = f.GetInt64("seed")
			}
			rows, err := fake.New(seed, locale).Rows(fields, count)
			if err != nil {
				return output.Usage("FAKE_FIELD_INVALID", err.Error(), "Xem danh sách: foxkit fake --list")
			}
			keys := fake.Keys(fields)
			if p.JSON {
				return p.Success(map[string]any{"count": count, "seed": seed, "rows": rows}, nil)
			}
			w := c.OutOrStdout()
			str := func(v any) string { return fmt.Sprint(v) }
			switch format {
			case "ndjson":
				for _, r := range rows {
					if err := p.Event(r); err != nil {
						return output.Internal(err)
					}
				}
			case "json":
				return printer(c).Success(rows, nil)
			case "csv":
				if bom {
					if _, err := io.WriteString(w, "\xef\xbb\xbf"); err != nil {
						return output.Internal(err)
					}
				}
				cw := csv.NewWriter(w)
				_ = cw.Write(keys) // lỗi ghi được Flush/Error báo ở dưới
				for _, r := range rows {
					rec := make([]string, len(r.Vals))
					for i, v := range r.Vals {
						rec[i] = str(v)
					}
					_ = cw.Write(rec)
				}
				cw.Flush()
				return cw.Error()
			default:
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				up := make([]string, len(keys))
				for i, k := range keys {
					up[i] = strings.ToUpper(k)
				}
				fmt.Fprintln(tw, strings.Join(up, "\t"))
				for _, r := range rows {
					cells := make([]string, len(r.Vals))
					for i, v := range r.Vals {
						cells[i] = str(v)
					}
					fmt.Fprintln(tw, strings.Join(cells, "\t"))
				}
				return tw.Flush()
			}
			return nil
		},
	}
	f := c.Flags()
	f.Int("count", 10, "Số dòng (1-100000)")
	f.String("format", "table", "table|json|csv|ndjson")
	f.Int64("seed", 0, "Seed (cùng seed → cùng kết quả)")
	f.String("locale", "vi", "vi|en")
	f.Bool("bom", false, "CSV có BOM (cho Excel đọc đúng tiếng Việt)")
	f.Bool("list", false, "Liệt kê field")
	rootCmd.AddCommand(c)
}
