package laralog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func sampleLog() string {
	var b strings.Builder
	b.WriteString("[2026-11-16 18:00:00] local.INFO: Một dòng thôi {\"user\":1}\n")
	b.WriteString(`[2026-11-16 18:55:02] local.ERROR: SQLSTATE[HY000] [2002] Connection refused {"exception":"[object] (Illuminate\\Database\\QueryException(code: 2002): SQLSTATE[HY000] [2002] Connection refused at D:\\Projects\\shop\\vendor\\laravel\\framework\\src\\Illuminate\\Database\\Connection.php:825)` + "\n")
	b.WriteString("[stacktrace]\n")
	for i := 0; i < 30; i++ {
		switch i {
		case 12:
			b.WriteString(`#12 D:\\Projects\\shop\\app\\Http\\Controllers\\OrderController.php(42): App\Http\Controllers\OrderController->store()` + "\n")
		case 13:
			b.WriteString(`#13 D:\\Projects\\shop\\app\\Services\\OrderService.php(88): x()` + "\n")
		default:
			fmt.Fprintf(&b, "#%d D:\\\\Projects\\\\shop\\\\vendor\\\\laravel\\\\framework\\\\src\\\\X.php(%d): y()\n", i, i)
		}
	}
	b.WriteString("\"}\n")
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&b, "[2026-11-16T19:%02d:00+07:00] local.ERROR: Lặp lại giống nhau {\"a\":1}\n", i)
	}
	b.WriteString("[2026-11-16 20:00:00] production.WARNING: Slow query 1200ms []") // không có \n cuối
	return b.String()
}

func write(t *testing.T, content string) string {
	p := filepath.Join(t.TempDir(), "laravel.log")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseAndMerge(t *testing.T) {
	es, err := ReadLast([]string{write(t, sampleLog())}, Filter{Dir: `D:\Projects\shop`}, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 4 { // INFO, ERROR sql, 3 ERROR gộp thành 1, WARNING
		t.Fatalf("số entry = %d: %+v", len(es), es)
	}
	sqlErr := es[1]
	if sqlErr.Level != "ERROR" || sqlErr.Exception != `Illuminate\Database\QueryException` || !strings.HasSuffix(sqlErr.File, "vendor/laravel/framework/src/Illuminate/Database/Connection.php:825") {
		t.Errorf("exception: %+v", sqlErr)
	}
	if strings.Contains(sqlErr.Message, "exception") || !strings.HasPrefix(sqlErr.Message, "SQLSTATE[HY000]") {
		t.Errorf("message phải bỏ context JSON: %q", sqlErr.Message)
	}
	if len(sqlErr.Trace) != 2 || !strings.Contains(sqlErr.Trace[0], "app/Http/Controllers/OrderController.php(42)") || strings.Contains(sqlErr.Trace[0], "D:") {
		t.Errorf("trace rút gọn (chỉ frame app/, đường dẫn tương đối): %v", sqlErr.Trace)
	}
	if es[2].Count != 3 || es[2].Message != "Lặp lại giống nhau" {
		t.Errorf("gộp entry giống nhau: %+v", es[2])
	}
	if es[3].Level != "WARNING" || es[3].Env != "production" || es[3].Message != "Slow query 1200ms" {
		t.Errorf("entry cuối không có \\n: %+v", es[3])
	}
	full, _ := ReadLast([]string{write(t, sampleLog())}, Filter{Full: true}, 50)
	if len(full[1].Trace) != 30 {
		t.Errorf("--full phải đủ 30 frame, got %d", len(full[1].Trace))
	}
}

func TestFilters(t *testing.T) {
	p := write(t, sampleLog())
	es, _ := ReadLast([]string{p}, Filter{MinLevel: "warning"}, 50)
	for _, e := range es {
		if e.Level == "INFO" {
			t.Fatal("--level warning phải bỏ INFO")
		}
	}
	if len(es) != 3 {
		t.Errorf("warning+ = %d", len(es))
	}
	es, _ = ReadLast([]string{p}, Filter{Grep: "sqlstate"}, 50)
	if len(es) != 1 {
		t.Errorf("grep: %d", len(es))
	}
	es, _ = ReadLast([]string{p}, Filter{}, 2)
	if len(es) != 2 || es[1].Level != "WARNING" {
		t.Errorf("--last 2 lấy 2 entry mới nhất theo thứ tự cũ→mới: %+v", es)
	}
	since := time.Date(2026, 11, 16, 19, 30, 0, 0, time.Local)
	es, _ = ReadLast([]string{p}, Filter{Since: since}, 50)
	if len(es) != 1 || es[0].Level != "WARNING" {
		t.Errorf("--since: %+v", es)
	}
}

func TestMultipleFilesNewestFirst(t *testing.T) {
	d := t.TempDir()
	a := filepath.Join(d, "laravel-2026-11-16.log")
	b := filepath.Join(d, "laravel-2026-11-15.log")
	os.WriteFile(a, []byte("[2026-11-16 10:00:00] local.ERROR: hôm nay\n"), 0o644)
	os.WriteFile(b, []byte("[2026-11-15 10:00:00] local.ERROR: hôm qua\n"), 0o644)
	es, err := ReadLast([]string{a, b}, Filter{}, 5)
	if err != nil || len(es) != 2 || es[0].Message != "hôm qua" || es[1].Message != "hôm nay" {
		t.Fatalf("%+v %v", es, err)
	}
}

func TestReadsOnlyTheTail(t *testing.T) {
	if testing.Short() {
		t.Skip("tạo file lớn")
	}
	p := filepath.Join(t.TempDir(), "big.log")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	line := "[2026-11-16 10:00:00] local.INFO: " + strings.Repeat("x", 200) + "\n"
	chunk := strings.Repeat(line, 5000)
	for i := 0; i < 100; i++ { // ~100MB
		f.WriteString(chunk)
	}
	f.WriteString("[2026-11-16 23:59:59] local.ERROR: cuối file\n")
	f.Close()
	start := time.Now()
	es, err := ReadLast([]string{p}, Filter{MinLevel: "error"}, 1)
	if err != nil || len(es) != 1 || es[0].Message != "cuối file" {
		t.Fatalf("%+v %v", es, err)
	}
	if time.Since(start) > 2*time.Second {
		t.Errorf("phải đọc ngược gần như tức thì, mất %v", time.Since(start))
	}
}

func TestFollowSeesAppendedAndRotation(t *testing.T) {
	p := write(t, "[2026-11-16 10:00:00] local.INFO: cũ\n")
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var got []string
	done := make(chan error, 1)
	go func() {
		done <- Follow(ctx, func() string { return p }, Filter{}, func(e Entry) { mu.Lock(); got = append(got, e.Message); mu.Unlock() })
	}()
	time.Sleep(700 * time.Millisecond)
	f, _ := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString("[2026-11-16 10:01:00] local.ERROR: mới 1\n[stacktrace]\n#0 x\n")
	f.Close()
	time.Sleep(1200 * time.Millisecond)
	os.WriteFile(p, []byte("[2026-11-16 11:00:00] local.ERROR: sau xoay vòng\n"), 0o644) // file nhỏ lại
	time.Sleep(1200 * time.Millisecond)
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(got, "|") != "mới 1|sau xoay vòng" {
		t.Fatalf("got %v", got)
	}
}
