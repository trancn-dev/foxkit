package lock

import (
	"testing"
	"unsafe"
)

// RM_PROCESS_INFO theo tài liệu Microsoft: 12 (RM_UNIQUE_PROCESS) + 512 + 128 + 4*4 = 668 byte.
func TestRmStructLayout(t *testing.T) {
	if got := unsafe.Sizeof(rmUniqueProcess{}); got != 12 {
		t.Errorf("rmUniqueProcess = %d, cần 12", got)
	}
	if got := unsafe.Sizeof(rmProcessInfo{}); got != 668 {
		t.Errorf("rmProcessInfo = %d, cần 668", got)
	}
}
