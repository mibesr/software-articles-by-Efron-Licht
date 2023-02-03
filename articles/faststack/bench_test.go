package ginex

import (
	"fmt"
	"testing"
)

var _res []byte

func BenchmarkFastStack(b *testing.B) {
	b.ReportAllocs()
	for i := 8; i <= 2048; i *= 2 {
		b.Run(fmt.Sprintf("depth=%d", 2*i), func(b *testing.B) { // we have a call to Ping(i) and Pong(i) for each i, so depth is 2*i
			for j := 0; j < b.N; j++ {
				b.StopTimer() // we don't want to count the time spent in ping-pong, just stack.
				_res = Ping(b, FastStack, i)
			}
		})
	}
}

func BenchmarkSlowStack(b *testing.B) {
	b.ReportAllocs()
	for i := 8; i <= 2048; i *= 2 {
		b.Run(fmt.Sprintf("depth=%04d", 2*i), func(b *testing.B) { // we have a call to Ping(i) and Pong(i) for each i, so depth is 2*i
			for j := 0; j < b.N; j++ {
				b.StopTimer() // we don't want to count the time spent in ping-pong, just stack.
				_res = Ping(b, SlowStack, i)
			}
		})
	}
}
