//go:build linux

package meta

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func OpenFileHandles() (int, error) {
	dir, err := os.ReadDir("/proc/self/fd/%d")
	return len(dir), err
}

var memRE = regexp.MustCompile(`\w+:\s+(\d+) kB`) // get it?
func MemInfo() (
	// in kB
	mi struct{ Total, Free, Available, Buffers, Cached int },
	err error,
) {
	b, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return mi, err
	}
	lines := strings.Split(string(b), "\n")
	for i, p := range []*int{&mi.Total, &mi.Free, &mi.Available, &mi.Buffers, &mi.Cached} {
		m := memRE.FindStringSubmatch(lines[i])
		*p, err = strconv.Atoi(m[1])
		if err != nil {
			return mi, fmt.Errorf("parsing %q: %w", lines[0], err)
		}
	}
	return mi, nil
}
