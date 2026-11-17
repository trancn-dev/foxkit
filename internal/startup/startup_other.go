//go:build !windows

package startup

func List() ([]Item, []string, error) { return nil, nil, ErrUnsupported }
