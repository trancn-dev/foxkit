package cmd

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"foxkit/internal/csvx"
	"foxkit/internal/fsutil"
	"foxkit/internal/output"

	"github.com/spf13/cobra"
)

const csvLimit = 200 << 20

func trunc(s string) string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", "↵")
	if utf8.RuneCountInString(s) > 40 {
		return string([]rune(s)[:39]) + "…"
	}
	return s
}

func delimOf(s string) (rune, bool) {
	switch s {
	case "auto", "":
		return 0, true
	case ",":
		return ',', true
	case ";":
		return ';', true
	case "tab", "\\t":
		return '\t', true
	case "|":
		return '|', true
	}
	return 0, false
}

func csvError(err error) error {
	var ce *csvx.ColError
	var we *csvx.WhereError
	switch {
	case errors.Is(err, csvx.ErrEncoding):
		return output.Fail("CSV_ENCODING", "File không phải UTF-8", `Mở bằng Excel → Save As → "CSV UTF-8 (Comma delimited)"`)
	case errors.As(err, &ce):
		return output.Usage("CSV_COL_NOT_FOUND", fmt.Sprintf("Không có cột %q", ce.Name), "Các cột có: "+strings.Join(ce.Have, ", "))
	case errors.As(err, &we):
		return output.Usage("CSV_WHERE_INVALID", we.Error(), `Ví dụ: --where "city=Hà Nội,age>18"`)
	}
	return output.Internal(err)
}

func init() {
	c := &cobra.Command{
		Use:   "csv <file|->",
		Short: "Xem, lọc, thống kê, đổi CSV ↔ JSON",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			f := c.Flags()
			delimStr, _ := f.GetString("delim")
			noHeader, _ := f.GetBool("no-header")
			cols, _ := f.GetStringSlice("cols")
			where, _ := f.GetString("where")
			head, _ := f.GetInt("head")
			tail, _ := f.GetInt("tail")
			stats, _ := f.GetBool("stats")
			to, _ := f.GetString("to")
			out, _ := f.GetString("out")
			force, _ := f.GetBool("force")
			typed, _ := f.GetBool("typed")
			if len(args) == 0 {
				return output.Usage("CSV_NO_INPUT", "Thiếu file CSV", "Ví dụ: foxkit csv users.csv")
			}
			delim, ok := delimOf(delimStr)
			if !ok {
				return output.Usage("ARGS_INVALID", "--delim phải là auto, ',', ';', tab hoặc '|'", "")
			}
			if to != "" && to != "json" && to != "ndjson" && to != "csv" && to != "tsv" {
				return output.Usage("ARGS_INVALID", "--to phải là json, ndjson, csv hoặc tsv", "")
			}
			if head < 0 || tail < 0 {
				return output.Usage("ARGS_INVALID", "--head/--tail không được âm", "")
			}
			var data []byte
			var err error
			if args[0] == "-" {
				data, err = readStdin(c, csvLimit, "200MB")
			} else {
				st, serr := os.Stat(args[0])
				if serr != nil {
					return output.FileNotFound(args[0])
				}
				if st.Size() > csvLimit {
					return output.InputTooLarge("200MB")
				}
				data, err = os.ReadFile(args[0])
			}
			if err != nil {
				return err
			}
			tb, err := csvx.Read(bytes.NewReader(data), delim, noHeader)
			if err != nil {
				return csvError(err)
			}
			if where != "" {
				if tb, err = tb.Where(where); err != nil {
					return csvError(err)
				}
			}
			if len(cols) > 0 {
				if tb, err = tb.Select(cols); err != nil {
					return csvError(err)
				}
			}
			total := len(tb.Rows)
			p := printer(c)

			if stats {
				st := tb.Stats()
				return p.Success(map[string]any{"columns": st, "total": total}, func(w io.Writer) error {
					tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
					fmt.Fprintln(tw, "CỘT\tKIỂU\tDÒNG\tRỖNG\tDUY NHẤT\tMIN\tMAX")
					for _, s := range st {
						fmt.Fprintf(tw, "%s\t%s\t%d\t%d\t%s\t%s\t%s\n", s.Name, s.Kind, s.Count, s.Empty, s.Unique, s.Min, s.Max)
					}
					return tw.Flush()
				})
			}

			rows := tb.Rows
			switch {
			case tail > 0 && len(rows) > tail: // --tail ưu tiên hơn --head mặc định
				rows = rows[len(rows)-tail:]
			case tail == 0 && head > 0 && len(rows) > head:
				rows = rows[:head]
			}
			shown := &csvx.Table{Header: tb.Header, Rows: rows, Delim: tb.Delim, HadBOM: tb.HadBOM}
			if to != "" { // xuất toàn bộ (đã lọc), không cắt --head
				shown.Rows = tb.Rows
			}

			if to != "" {
				var buf bytes.Buffer
				switch to {
				case "json":
					b, err := jsonIndent(shown.Objects(typed))
					if err != nil {
						return output.Internal(err)
					}
					buf.Write(b)
					buf.WriteByte('\n')
				case "ndjson":
					for _, o := range shown.Objects(typed) {
						b, err := o.MarshalJSON()
						if err != nil {
							return output.Internal(err)
						}
						buf.Write(b)
						buf.WriteByte('\n')
					}
				default:
					cw := csv.NewWriter(&buf)
					cw.Comma = ','
					if to == "tsv" {
						cw.Comma = '\t'
					}
					cw.UseCRLF = false // luôn LF
					_ = cw.Write(shown.Header)
					_ = cw.WriteAll(shown.Rows) // lỗi ghi vào buffer không xảy ra
					cw.Flush()
				}
				if out != "" {
					if fsutil.Exists(out) && !force {
						return output.Fail("FILE_EXISTS", "File đã tồn tại: "+out, "Thêm --force để ghi đè")
					}
					if err := os.WriteFile(out, buf.Bytes(), 0o644); err != nil {
						return output.Internal(err)
					}
					return p.Success(map[string]any{"written": out, "rows": len(shown.Rows)}, func(w io.Writer) error {
						_, err := fmt.Fprintf(w, "Đã ghi %d dòng ra %s\n", len(shown.Rows), out)
						return err
					})
				}
				_, err := c.OutOrStdout().Write(buf.Bytes())
				return err
			}

			warnings := tb.Warnings
			if warnings == nil {
				warnings = []string{}
			}
			return p.Success(map[string]any{"header": shown.Header, "rows": shown.Rows, "total": total, "delim": string(tb.Delim), "bom": tb.HadBOM, "warnings": warnings}, func(w io.Writer) error {
				tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
				fmt.Fprintln(tw, strings.Join(shown.Header, "\t"))
				for _, r := range shown.Rows {
					cells := make([]string, len(r))
					for i, v := range r {
						cells[i] = trunc(v)
					}
					fmt.Fprintln(tw, strings.Join(cells, "\t"))
				}
				if err := tw.Flush(); err != nil {
					return err
				}
				if more := total - len(shown.Rows); more > 0 {
					fmt.Fprintf(w, "… %s dòng nữa (dùng --head 0 để xem hết)\n", fsutil.FormatInt(more))
				}
				for _, wn := range warnings {
					fmt.Fprintf(c.ErrOrStderr(), "Cảnh báo: %s\n", wn)
				}
				return nil
			})
		},
	}
	f := c.Flags()
	f.String("delim", "auto", "auto, ',', ';', tab hoặc '|'")
	f.Bool("no-header", false, "Dòng đầu là dữ liệu")
	f.StringSlice("cols", nil, "Cột (tên hoặc số thứ tự 1-based)")
	f.String("where", "", `Điều kiện lọc, ví dụ "city=Hà Nội,age>18"`)
	f.Int("head", 20, "Số dòng đầu (0 = tất cả)")
	f.Int("tail", 0, "Số dòng cuối")
	f.Bool("stats", false, "Thống kê cột")
	f.String("to", "", "json|ndjson|csv|tsv")
	f.String("out", "", "Ghi ra file")
	f.Bool("force", false, "Ghi đè file --out")
	f.Bool("typed", false, "--to json: số/bool thành kiểu thật")
	rootCmd.AddCommand(c)
}
