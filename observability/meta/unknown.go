//go:build !linux

package meta

func OpenFileHandles() (int, error) {
	return 0, nil
}
func ignoreErr[T any](t T, _ error) T { return t }

var memRE = regexp.MustCompile(`\w+:\s+(\d+) kB`) // get it?
// MemInfo returns a struct with all zeroes.
func MemInfo() (
	// in kB
	mi struct{ Total, Free, Available, Buffers, Cached int },
	err error,
) {
	return mi, nil
}
