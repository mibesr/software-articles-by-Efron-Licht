package ginex

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
)

var (
	localSourceOnce     sync.Once
	localSourceNotFound bool
)

// local source is not always available. for example, the executable may be running on a system without the source
// or the -trimpath buildflag could have been provided to go tool.
// if we can't find the source for THIS file, we are unlikely to be able to find it for any file.
func localSourceUnavailable() bool {
	localSourceOnce.Do(func() {
		_, file, _, _ := runtime.Caller(3)
		if _, err := os.Lstat(file); err != nil {
			localSourceNotFound = true
		}

	})
	return localSourceNotFound
}

type debugInfo struct {
	*runtime.Frame
	Source string // line of code where the call appeared
	Depth  int
}

// FastStack returns a formatted FastStack frame, skipping debug frames
func FastStack(skip int) (formatted []byte) {
	// grab the stack frame
	pc := make([]uintptr, 64)
	n := runtime.Callers(skip, pc)
	if n == 0 { // no callers: e.g, skip > len(callstack).
		return nil
	}
	pc = pc[:n] // pass only valid pcs to runtime.Caller
	frames := runtime.CallersFrames(pc)
	// allocate a 4KiB reusable buffer, exactly once. we will use this both to read the input and format the output.
	buf := make([]byte, 0, 4096)
	if localSourceUnavailable() {
		// fast path: just format the frames in the order they occur without looking up the source.
		out := bytes.NewBuffer(buf)
		for {
			frame, more := frames.Next()
			fmt.Fprintf(out, "%s:%d (0x%x)\t%s:%s\n", frame.File, frame.Line, frame.PC, trimFunction(frame.Function), "")
			if !more {
				return out.Bytes()
			}
		}
	}
	// slow path: at least some local source is available, so we want to populate the debuginfo with the lines of code that appear in the stack  trace.
	di := make([]debugInfo, 0, n)
	for depth := 0; ; depth++ {
		frame, more := frames.Next()
		di = append(di, debugInfo{Frame: &frame, Depth: depth})
		if !more {
			break
		}
	}

	// group the debuginfo by file and line: we'll resort them by depth later.
	sort.Slice(di, func(i, j int) bool {
		return di[i].File < di[j].File || (di[i].File == di[j].File && di[i].Line < di[j].Line)
	})
	// populate debug info with source.
	func() {
		line := 0
		lastFile := di[0].File
		f, err := os.Open(lastFile)
		defer func() {
			// scanner.Scan() can panic if it hits too many newlines.
			// in that case, we immediately abandon trying to get more sourceInfo and just write as much as we can.
			if p := recover(); p != nil {
				log.Println("panic while formatting stack: too many empty lines in sourcefile?", p, debug.Stack())
				f.Close()
			}

		}()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(buf[:0], bufio.MaxScanTokenSize)
		for i := range di {
			if di[i].File != lastFile {
				if err == nil {
					f.Close()
				}
				// reset: we're at the beginning of a new file
				line = 0
				f, err = os.Open(di[i].File)
				if err != nil {
					continue
				}
				scanner = bufio.NewScanner(f)
				scanner.Buffer(buf[:0], bufio.MaxScanTokenSize)
			}
			if err != nil {
				continue
			}
			// it's possible that we have multiple calls to the same function in the stack, such as during recursion,
			// so we check that we haven't gone past BEFORE we advance the scanner.
			for ; line < di[i].Line; line++ {
				scanner.Scan()
			}
			di[i].Source = scanner.Text()

			i++
		}
		_ = f.Close()
	}()

	// put the debuginfo back in depth-first order
	sort.Slice(di, func(i, j int) bool { return di[i].Depth < di[j].Depth })
	// format it all into the buffer. we're safe to reuse buf, since we're done with all the scanners.
	out := bytes.NewBuffer(buf[:0])
	for i := range di {
		fmt.Fprintf(out, "%s:%d (0x%x)\t%s:%s\n", di[i].File, di[i].Line, di[i].PC, trimFunction(di[i].Function), strings.TrimSpace(di[i].Source))

	}

	return out.Bytes()
}

func trimFunction(name string) string {
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contain dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := strings.LastIndex(name, "/"); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	const centerDot = "·" //
	if period := strings.Index(name, "."); period >= 0 {
		name = name[period+1:]
	}
	return strings.ReplaceAll(name, centerDot, ".")
}
