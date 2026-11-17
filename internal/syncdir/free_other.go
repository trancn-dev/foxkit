//go:build !windows

package syncdir

// freeBytes: chưa hỗ trợ nơi khác Windows nên bỏ qua kiểm tra dung lượng.
func freeBytes(string) (uint64, bool) { return 0, false }
