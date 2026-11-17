package winexec

import (
	"context"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Proc struct {
	PID      int
	Name     string
	MemBytes int64
}

// Tasklist: tasklist /fo csv /nh
func Tasklist(ctx context.Context) ([]Proc, error) {
	out, err := Run(ctx, 15*time.Second, "tasklist", "/fo", "csv", "/nh")
	if err != nil {
		return nil, err
	}
	return ParseTasklist(string(out))
}

// ProcName tra tên process theo PID; "" nếu không có.
func ProcName(ctx context.Context, pid int) (string, error) {
	out, err := Run(ctx, 15*time.Second, "tasklist", "/fi", fmt.Sprintf("PID eq %d", pid), "/fo", "csv", "/nh")
	if err != nil {
		return "", err
	}
	procs, err := ParseTasklist(string(out))
	if err != nil || len(procs) == 0 {
		return "", err
	}
	return procs[0].Name, nil
}

// ParseTasklist đọc CSV của tasklist; dòng không phải CSV 5 cột (vd thông báo "No tasks") bị bỏ qua.
func ParseTasklist(s string) ([]Proc, error) {
	r := csv.NewReader(strings.NewReader(s))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	procs := make([]Proc, 0, len(rows))
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil {
			continue
		}
		procs = append(procs, Proc{PID: pid, Name: row[0], MemBytes: parseKB(row[4])})
	}
	return procs, nil
}

// parseKB: "412,345 K" hoặc "412.345 K" → byte. Bỏ mọi ký tự không phải số.
func parseKB(s string) int64 {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	n, _ := strconv.ParseInt(b.String(), 10, 64) // rỗng → 0, chấp nhận
	return n * 1024
}
