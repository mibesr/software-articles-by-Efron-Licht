// printpair.go
// parse a pair of UUIDs into their canonical forms, then print them byte-by-byte as a markdown table
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
)

func main() {
	if len(os.Args) != 3 {
		log.Print("USAGE: printpair UUID UUID")
		log.Print("EXAMPLE: printpair b5978cb8-ec12-42a7-8b8a-17fdf2cc812e c31727f1-a46c-45ea-a177-37b1f3b54611")
		log.Fatal("expected exactly two command-line args")
	}
	csharp, golang := uuid.MustParse(os.Args[1]), uuid.MustParse(os.Args[2])
	fmt.Printf("|i|c#|golang|\n")
	for i := 0; i < 16; i++ {
		// the %x formatting verb means "format this as hexidecimal"
		// the %02x means "and pad it with zeroes to length 2, if it's shorter than that."
		fmt.Printf("|%x|%02x|%02x|\n", i, csharp[i], golang[i])
	}
}
