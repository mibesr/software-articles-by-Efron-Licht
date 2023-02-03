// build-table maps the relationship between two UUIDS.
package main // buildtable.go

import (
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
)

func main() {
	if len(os.Args) != 3 {
		log.Print("USAGE: buildtable UUID UUID")
		log.Print("EXAMPLE: buildtable b5978cb8-ec12-42a7-8b8a-17fdf2cc812e c31727f1-a46c-45ea-a177-37b1f3b54611")
		log.Fatal("expected exactly two command-line args")
	}
	csharp, golang := uuid.MustParse(os.Args[1]), uuid.MustParse(os.Args[2])
	var c2g, g2c [16]byte
	for i := range csharp {
		for j := range golang {
			if csharp[i] == golang[j] {
				c2g[i] = byte(j)
				g2c[j] = byte(i)
			}
		}
	}
	fmt.Printf("|i|c2g|g2c|\n")
	fmt.Printf("|---|---|---|\n")

	for i := 0; i < 16; i++ {
		fmt.Printf("|%x|%02x|%02x|\n", i, c2g[i], g2c[i])
	}
}
