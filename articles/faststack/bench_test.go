package ginex

import (
	"fmt"
	"testing"
)

var _res []byte

func BenchmarkPing(b *testing.B) {
	b.ReportAllocs()
	for i := 8; i <= 2048; i *= 2 {
		b.Run(fmt.Sprintf("depth=%d", 2*i), func(b *testing.B) { // we have a call to Ping(i) and Pong(i) for each i, so depth is 2*i
			b.Run("faststack", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					b.StopTimer() // we don't want to count the time spent in ping-pong, just stack.
					_res = Ping(b, FastStack, i)
				}
			})
			b.Run("slowstack", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					// we don't want to count the time spent in ping-pong, just stack.
					b.StopTimer() // we don't want
					_res = Ping(b, SlowStack, i)
				}
			})
		})
	}
}
