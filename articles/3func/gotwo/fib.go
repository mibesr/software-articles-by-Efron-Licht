// go2-fib prints the first 10 fibonacci numbers.
package main

// global variables shared by everything.
var i, prev, cur, tmp int

func main() {
	// initialize
	prev = 0
	cur = 1
loop:
	println(cur)
	tmp = cur // no multiple assignment, so use a temporary variable as scratch space.
	cur = prev + cur
	prev = tmp
	i = i + 1
	if i < 10 {
		goto loop
	}
}
