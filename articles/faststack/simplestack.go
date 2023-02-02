package ginex

import (
	"fmt"
	"io"
	"runtime"
)

func WriteStack(w io.Writer, skip int) (int, error) {
	// grab the stack frame
	pc := make([]uintptr, 64)
	n := runtime.Callers(skip, pc)
	if n == 0 { // no callers: e.g, skip > len(callstack).
		return 0, nil
	}
	pc = pc[:n] // pass only valid pcs to runtime.Caller
	frames := runtime.CallersFrames(pc)
	total := 0
	for {
		frame, more := frames.Next()
		n, err := fmt.Fprintf(w, "%s:%d (0x%x)\t%s:%s\n", frame.File, frame.Line, frame.PC, trimFunction(frame.Function), "")
		total += n
		if err != nil {
			return total, err
		}
		if !more {
			return total, nil
		}
	}
	panic("unreachable!")
}
