// fmtbench
package main // fmtbench/main.go

import (
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type result struct {
	name                    string
	runs, ns, bytes, allocs float64
}

func mustTrimPrefix(s, prefix string) string {
	_, after, ok := strings.Cut(s, prefix)
	if !ok {
		must(after, fmt.Errorf("expected input to have a prefix %q", prefix))
	}
	return after
}
func main() {
	input := strings.TrimSpace(string(must(io.ReadAll(os.Stdin))))
	lines := strings.Split(input, "\n")
	goarch := mustTrimPrefix(lines[0], "goarch: ")
	goos := mustTrimPrefix(lines[1], "goos: ")
	pkg := mustTrimPrefix(lines[2], "pkg: ")
	fmt.Printf("## benchmark %s: %s/%s\n", pkg, goos, goarch)
	fmt.Println(`|name|runs|ns/op|%/max|bytes|%/max|allocs|%/max|`)
	fmt.Println(`|---|---|---|---|---|---|---|---|`)
	// get results and min/max
	var results []result
	var maxNS, maxBytes, maxAllocs float64
	{
		// thank you
		var re = regexp.MustCompile(`Benchmark(.+)(?:-\w)\s+(\d+)\s+(.+)ns/op\s+(\d+) B/op\s+(\d+)`)

		for _, line := range lines {
			match := re.FindStringSubmatch(line)
			if match == nil {
				continue
			}
			atof := func(i int) float64 { return must(strconv.ParseFloat(strings.TrimSpace(match[i]), 64)) }
			res := result{name: match[1], runs: atof(2), ns: atof(3), bytes: atof(4), allocs: atof(5)}
			results = append(results, res)
			maxNS, maxBytes, maxAllocs = math.Max(maxNS, res.ns), math.Max(maxBytes, res.bytes), math.Max(maxAllocs, res.allocs)
		}
	}
	sort.SliceStable(results, func(i, j int) bool { return results[i].ns < results[j].ns })
	for _, res := range results {
		fmt.Printf("|%s|%.3g|%.3g|%0.3g|%.3g|%0.3g|%.3g|%0.3g|\n", res.name, res.runs, res.ns, (res.ns/maxNS)*100, res.bytes, (res.bytes/maxBytes)*100, res.allocs, (res.allocs/maxAllocs)*100)
	}
}

func must[T any](t T, err error) T {
	if err != nil {
		log.Print("unexpected error")
		log.Print("USAGE:  go test -bench=. -benchmem DIR | bench")
		log.Fatal(err)
	}
	return t
}
