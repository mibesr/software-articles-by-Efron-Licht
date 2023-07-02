// fmtbench
package main // fmtbench/main.go

import (
	"flag"
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

var sortBy = flag.String("sort-by", "none", "sort criteria: options 'none' 'allocs' 'name', 'runtime'")

func main() {
	flag.Parse()
	switch strings.ToLower(*sortBy) {
	case "none", "allocs", "name", "runtime":
		// nop
	default:
		flag.Usage()
		log.Fatalf("unexpected value %q for flag -sortby", *sortBy)
	}
	input := strings.TrimSpace(string(must(io.ReadAll(os.Stdin))))
	lines := strings.Split(input, "\n")
	goos := strings.TrimPrefix(lines[1], "goarch: ")
	goarch := strings.TrimPrefix(lines[0], "goos: ")
	pkg := strings.TrimPrefix(lines[2], "pkg: ")
	fmt.Printf("## benchmarks %s: %s/%s\n", pkg, goos, goarch)
	fmt.Println(`|name|runs|ns/op|%/max|bytes|%/max|allocs|%/max|`)
	fmt.Println(`|---|---|---|---|---|---|---|---|`)
	// get results and min/max
	type result struct {
		name                    string
		runs, ns, bytes, allocs float64
	}
	//
	var results []result
	var maxNS, maxBytes, maxAllocs float64
	{ // parse results
		re := regexp.MustCompile(`Benchmark(.+)\s+(\d+)\s+(.+)ns/op\s+(\d+) B/op\s+(\d+)`)

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

	{ // sort results
		var less func(i, j int) bool
		switch *sortBy {
		case "none":
			goto PRINT
		case "allocs":
			less = func(i, j int) bool { return results[i].allocs < results[j].allocs }
		case "name":
			less = func(i, j int) bool { return results[i].name < results[j].name }
		case "runtime":
			less = func(i, j int) bool { return results[i].ns < results[j].ns }
		}
		sort.Slice(results, less)
	}
PRINT:
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
