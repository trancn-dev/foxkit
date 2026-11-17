//go:build !windows

package disk

func List(bool, float64) ([]Drive, error) { return nil, ErrUnsupported }
