//go:build !windows

package lock

func Find(paths []string, maxFiles int) (Result, error) {
	return Result{Paths: paths, Holders: []Holder{}}, ErrUnsupported
}
