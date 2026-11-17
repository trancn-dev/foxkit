package sysinfo

import (
	"testing"
	"unsafe"
)

// MEMORYSTATUSEX theo tài liệu Microsoft là 64 byte; sai layout C thì GlobalMemoryStatusEx ghi lố hoặc trả rác.
func TestMemoryStatusExLayout(t *testing.T) {
	if got := unsafe.Sizeof(memoryStatusEx{}); got != 64 {
		t.Fatalf("unsafe.Sizeof(memoryStatusEx) = %d, cần 64", got)
	}
}
