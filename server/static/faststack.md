
# optimizing gin's stack

## gin's panic handler

The popular go web framework Gin has a middleware that allows you to recover from and log panics while serving HTTP.

Here's an example program that will panic whenever we hit the route GET /panic

```go
package main

import (
   "fmt"
   "net/http"

   "github.com/gin-gonic/gin"
)

func main() {
   engine := gin.New()
   engine.Use(gin.Recovery())
   engine.GET("/panic", func(c *gin.Context) {fmt.Fprintf(c.Writer, "%s", f())})
   http.ListenAndServe(":8080", engine)
}
func f() string {
    panic("this function panics!")
}
```

We run the program:

```sh
go run main.go
```

and in another shell provoke the panic:

```sh
curl http://localhost:8080
```

Getting this nicely formatted result:

```
2023/01/17 13:45:01 [Recovery] 2023/01/17 - 13:45:01 panic recovered:
GET /panic HTTP/1.1
Host: localhost:8080
Accept: */*
User-Agent: curl/7.81.0


this function panics!
/home/efron/go/src/gitlab.com/efronlicht/gin-ex/main.go:19 (0x7200a6)
        f: panic("this function panics!")
/home/efron/go/src/gitlab.com/efronlicht/gin-ex/main.go:14 (0x720094)
        main.func1: fmt.Fprintf(c.Writer, "%s", f())
/home/efron/go/pkg/mod/github.com/gin-gonic/gin@v1.8.2/context.go:173 (0x71a601)
        (*Context).Next: c.handlers[c.index](c)
/home/efron/go/pkg/mod/github.com/gin-gonic/gin@v1.8.2/recovery.go:101 (0x71a5ec)
        CustomRecoveryWithWriter.func1: c.Next()
/home/efron/go/pkg/mod/github.com/gin-gonic/gin@v1.8.2/context.go:173 (0x719470)
        (*Context).Next: c.handlers[c.index](c)
/home/efron/go/pkg/mod/github.com/gin-gonic/gin@v1.8.2/gin.go:616 (0x7190d8)
        (*Engine).handleHTTPRequest: c.Next()
/home/efron/go/pkg/mod/github.com/gin-gonic/gin@v1.8.2/gin.go:572 (0x718d9c)
        (*Engine).ServeHTTP: engine.handleHTTPRequest(c)
/usr/local/go/src/net/http/server.go:2947 (0x620a6b)
        serverHandler.ServeHTTP: handler.ServeHTTP(rw, req)
/usr/local/go/src/net/http/server.go:1991 (0x61d366)
        (*conn).serve: serverHandler{c.server}.ServeHTTP(w, w.req)
/usr/local/go/src/runtime/asm_amd64.s:1594 (0x4672e0)
        goexit: BYTE    $0x90   // NOP
```

As well as the usual elements of a stack trace: file, line, program counter, function name; Gin's recovery handler somehow gives you the **actual line of the source code** where that frame of the stack trace happened.

For programmers more used to dynamic languages, this seems pretty natural: you need the source code to run the file. But Go is a statically compiled language that doesn't carry around it's source code.

I had two thoughts when I saw this:

- "wow, cool!"
- "this is probably incredibly slow".

Let's take a look into the implementation to find out _how_ Gin does it, and see if we can do it better. When we're done, let's run some benchmarks to compare the solutions, and then talk about if this source-code magic is a good idea in the first place.

## Investigating Gin

Luckily, the stack trace itself gives us nearly all the clues we need:
> `/home/efron/go/pkg/mod/github.com/gin-gonic/gin@v1.8.2/recovery.go:101 (0x71a5ec)
        CustomRecoveryWithWriter.func1: c.Next()`

which contains a function `stack()`, which is simple and clearly documented. From now on, I'm going to call it `slowstack()`, and we'll call our eventual optimized version `faststack()`.
Starting at `skip`, we walk frame-by-frame up the stack.

for each frame, we use `runtime.Caller` to get the stack information for each one, using the `file` name to look up up a file with the same name on the host system.

We read the entire file into memory, caching the most-recent file, so multiple calls to the same function don't have to re-read the file, and print out the annotated stack frame.

```go
// stack returns a nicely formatted stack frame, skipping skip frames.
func stack(skip int) []byte { // from now on, called `slowstack()`
 // As we loop, we open files and read them. These variables record the currently
 // loaded file.
 var lines [][]byte
 var lastFile string
 for i := skip; ; i++ { // Skip the expected number of frames
  pc, file, line, ok := runtime.Caller(i)
  if !ok {
   break
  }
  // Print this much at least.  If we can't find the source, it won't show.
  fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
  if file != lastFile {
   data, err := os.ReadFile(file)
   if err != nil {
    continue
   }
   lines = bytes.Split(data, []byte{'\n'})
   lastFile = file
  }
  fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
 }
 return buf.Bytes()
}

func source(lines [][]byte, n int) {
 n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
 if n < 0 || n >= len(lines) {
  return dunno
 }
 return bytes.TrimSpace(lines[n])
}
// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte{ // body omitted
}

```

What are the performance limitations of `slowstack`?

- unnecessary work in the `runtime` package
- always reads a whole file instead of a single line
- can re-read the same file multiple times
- opens many file handles
- unbounded allocations
- always attemps to read files even if it's impossible

### unbounded memory usage

### unnecessary work in the `runtime` package

`runtime.Caller()` is a specalized invocation of runtime.CallersFrames for a single function: it ascends the callstack to `skip` using runtime magic, builds a `runtime.Frame` struct, and gives you a couple of that struct's fields:

Let's look at the source:

```go
type Frame struct {
 // PC is the program counter for the location in this frame.
 // For a frame that calls another frame, this will be the
 // program counter of a call instruction. Because of inlining,
 // multiple frames may have the same PC value, but different
 // symbolic information.
 PC uintptr

 // Func is the Func value of this call frame. This may be nil
 // for non-Go code or fully inlined functions.
 Func *Func

 // Function is the package path-qualified function name of
 // this call frame. If non-empty, this string uniquely
 // identifies a single function in the program.
 // This may be the empty string if not known.
 // If Func is not nil then Function == Func.Name().
 Function string

 // File and Line are the file name and line number of the
 // location in this frame. For non-leaf frames, this will be
 // the location of a call. These may be the empty string and
 // zero, respectively, if not known.
 File string
 Line int

 // Entry point program counter for the function; may be zero
 // if not known. If Func is not nil then Entry ==
 // Func.Entry().
 Entry uintptr
 // contains filtered or unexported fields
}
// package runtime
func Caller(skip int) (pc uintptr, file string, line int, ok bool) {
 rpc := make([]uintptr, 1)
 n := callers(skip+1, rpc[:])
 if n < 1 {
  return
 }
 frame, _ := CallersFrames(rpc).Next() // a runtime.Frame
 return frame.PC, frame.File, frame.Line, frame.PC != 0
}
```

`slowstack` then calls `function(pc)` on the retuned program counter to lookup the `*runtime.Func`, then get it's `Name`.

We could could simply grab the `(runtime.Frame).Function` field directly. Better yet, we can call `runtime.Callers()` and `runtime.CallersFrames()` exactly once and ascend the callstack by repeatedly calling `frames.Next()`, which avoids having to repeatedly jump up and down the stack.

```go
// stack returns a nicely formatted stack frame, skipping skip frames.
func stack_01(skip int) []byte {
    // grab the stack frame
    pc := make([]uintptr, 64)
    n := runtime.Callers(skip, pc)
    if n == 0 { // no callers: e.g, skip > len(callstack).
        return nil
    }
    pc = pc[:n] // pass only valid pcs to runtime.Caller
    buf := new(bytes.Buffer)
    frames := runtime.CallersFrames(pc)
    for {
        frame, more := frames.Next()
        fmt.Fprintf(buf, "%s:%d (0x%x)\n", frame.File, frame.Line, frame.PC)
    if file != lastFile {
        data, err := os.ReadFile(file)
        if err != nil {
            continue
        }
        lines = bytes.Split(data, []byte{'\n'})
        lastFile = file
         fmt.Fprintf(buf, "\t%s: %s\n", frame.Function, source(lines, line))
    }
    if !more {
        return buf.Bytes()
    }
    }
    panic("unreachable!)
}
```

This approach has one limitation: we have to specify a max stack size to pass to `runtime.Callers`. I think this is a _good thing_, since it places bounds on the resources this function can use, but it _is_ a limitation. You could mitigate this by making the max depth a parameter of the function and configuring, but 64 seems like a good number to me.

## always reads a whole file instead of a single line

 Even with an incredibly simple handler, like ours, `slowstack` reads 7 files into memory, totalling 276KiB.

- main.go (.3 KiB)
- gin/context.go (37 KiB)
- gin/recovery.go (6 Kib)
- gin/context.go again (37 KiB)
- gin/gin.go (23 KiB)
- net/http/server.go (114 KiB)
- asm_amd64.s (59KiB)

The longest lines of a `.go` or `.s` file are roughly 200 bytes, so this is using  roughly 1,300 times the memory it needs to.

## reading a line at a time

We can read one line at a time instead by using `bufio.Scanner`. We'll need a new scanner per file, but we can share a buffer between them.
This has two benefits: first, we allocate less memory. Secondly, we can _stop_ reading a file once we've hit the appropriate line without having to read to the end.

```go
func stack_02(skip int) []byte {
    // grab the stack frame
    pc := make([]uintptr, 64)
    n := runtime.Callers(skip, pc)
    if n == 0 { // no callers: e.g, skip > len(callstack).
        return nil
    }
    pc = pc[:n] // pass only valid pcs to runtime.Caller
    buf := new(bytes.Buffer)
    frames := runtime.CallersFrames(pc)
    scanBuf := make([]byte, 0, 256)
    FRAME:
    for {
        frame, more := frames.Next()
        fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
        f, err := os.Open(file)
        if err != nil {
            continue
        }
        f, err := os.Open(frame.File)
        if err != nil {
            continue FRAME
        }
        scanner := bufio.NewScanner(f)
        scanner.Buffer(scanBuf[:0], bufio.MaxScanTokenSize)
        var source []byte
        for i := 0; scanner.Scan() && i < Line; i++ {
            if i == line {
                fmt.Fprintf(buf, "\t%s: %s\n", function(pc),  bytes.TrimSpace(scanner.Bytes())
                f.Close()
                continue FRAME
            }
        }
        // hit EOF early:
        fmt.Fprintf(buf, "\t%s: %s\n", function(pc),  "???")
        f.Close()
    }
    return buf.Bytes()
}
```

Two caveats: while `stack_02` uses less memory _per file_ than `stack_01` or `slowstack` we still have to open the same file each time it appears on the callstack. `slowstack` had an optimization for a common case: if the same file appeared twice in a row on the callstack, it would re-use the memory it had previously read.

Second, `stack_02` has a hidden if unlikely bug: a `bufio.Scanner` can panic if it hits too many empty tokens in a row (in this case, dozens of newlines). In our next iteration, we'll wrap this in a `recover()` to protect ourselves from weird sourcefiles: we don't want to panic during a panic recovery handler! We'll handle this in our final solution.

## reading each file _exactly once_

While our solution uses less memory _per file_, we still have to open the same file each time it appears on the callstack. Pretty bad if we hit a recursive function:

```go

func FibStack(n int) int {
    if n < 0 {
        panic("expected n >= 0")
    }
    if n == 0 || n == 1 {
        fmt.Println(string(stack(0)))
        return 1
    } else {
        return FibStack(n-1) + FibStack(n-2)
    }
    default:
        return
    }
```

But `slowstack`'s solution was no prize, either: callstacks that rapidly bounce between the same files with at least one different file in-between get no savings from that approach.

We can build our own pathological example (we'll use this for benchmarking later)

```go
// ping.go
func PingStack(n int) []byte {
    if n < 0 {
        panic("expected n >= 0")
    }
    if n == 0 {
        return stack(0)
    }
    return Pong(n-1)
}
func Pong(n int) []byte {return Ping(n)}
```

We'd like to open each file exactly once, which means grouping our stack frames by filename. There's a few approaches you could use, but I prefer this one:
If we sort the frames by file-and-line, we can populate each frame with the source in exactly one pass thorough each file. We can then format the annotated frames.

This will require a new datastructure:

```go
type debugInfo struct {
        *runtime.Frame // hold a pointer rather than allocate a new one
        Source string // the line of source code holding the frame
        Depth int // the frame's original depth in the stack
}
```

This requires a few structural changes: rather than formatting the frames as we ascend the callstack, we put them all in a slice, populate them all, resort-them with their original order, and _then_ format them into our output buffer.

If there are few or no repeated files, this will use more peak memory than the previous approach,  but it's a solid solution that saves us from the pathological cases we outlined above. And since our minimal test case proves _any_ Gin handler will have some repeated cases, we always save _some_ work.

```go
func stack_03(skip int) (formatted []byte) {
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
    type debugInfo struct {
        *runtime.Frame
        Source string
        Depth int
    }
    // populate the debuginfo with the lines of code that appear in the stack  trace.
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
```

Pretty good! We're nearly done. When there are files to read, we read them efficiently.

## what if we can't read the files?

There's plenty of situations where we can't read the files at all. Off the top of my head:

- The executable was built with `go build -trimpath`.
- It's running in a binary without access to the source, like a minimal container.
- The running user doesn't have permission to access the source.

In this case, our implementation does a ton of work that's completely unneccessary, juggling around a ton of data just before it's inevitable failure.

If we had some kind of heuristic for "can read source files", we'd know whther or not it was worth the attempt. We could then build an fast-path for that case.

Here, we use a `sync.Once` to lazily-check whether or not we can access the file where we defined `faststack`():

```go
// local source is not always available. for example, the executable may be running on a system without the source
 // or the -trimpath buildflag could have been provided to go tool.
 // if we can't find the source for THIS file, we are unlikely to be able to find it for any file.  
var localSourceOnce sync.Once
var localSourceNotFound bool
func localSourceUnavailable() bool {
        localSourceOnce.Do(func() {
            _, file, _, _ := runtime.Caller(3)
            if _, err := os.Lstat(file); err != nil {
                localSourceNotFound = true
            }

        })
        return localSourceNotFound
    }
```

and then we can put it all together:

## finished faststack

```go
// stack returns a formatted stack frame, skipping debug frames
func faststack(skip int) (formatted []byte) {
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
        di = append(di, struct {
            *runtime.Frame
            Source string // line of code where the call appeared
            Depth  int
        }{Frame: &frame, Depth: depth})
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
```

## benchmarks

This behavior is hard to benchmark, since how it performs depends wildly on the structure of the callstack and actual files it's reading, which could be wildly different. Ideally, we'd collect a wide sample set of real Go code, insert calls to `faststack` and `slowstack`, and run it on dozens or hundreds of combinations of hardware, operating system, etc, etc.

I'm way too lazy for that and this article's already taken eight hours longer than I thought it would, so instead we'll make a couple simple synthetic benchmarks at a variety of stack depths and try and draw some conclusions.

We'll run the benchmarks on the same computer under four different circumstances:

linux (wsl), source code on SSD
linux (wsl), source code on SSD, compiled with `-trimpath`
windows, source on SSD
windows, source on HDD

Please see the [benchmark_results.md](benchmark_results.md) and the source code in [bench_test.go](bench_test.go), `[ping_test.go]`(ping_test.go), and `[pong_test.go]`(pong_test.go) for additional details.

Caveat: These benchmarks are somewhat unfair to `slowstack`, since they're constructed to attack a known worst-case for it's performance, and _should not be taken as definitive conclusions_.

Let's examine the first case:

## linux(wsl): SSD

BenchmarkPing/depth=1/faststack-24             14306      78653 ns/op     8849 B/op       92 allocs/op
BenchmarkPing/depth=1/slowstack-24             12033     105193 ns/op   220315 B/op      101 allocs/op
BenchmarkPing/depth=5/faststack-24             10000     101501 ns/op    10835 B/op      132 allocs/op
BenchmarkPing/depth=5/slowstack-24              7717     135690 ns/op   302127 B/op      157 allocs/op
BenchmarkPing/depth=25/faststack-24             6556     180071 ns/op    20765 B/op      332 allocs/op
BenchmarkPing/depth=25/slowstack-24             3907     313058 ns/op   719256 B/op      440 allocs/op
BenchmarkPing/depth=125/faststack-24            4456     260204 ns/op    44900 B/op      651 allocs/op
BenchmarkPing/depth=125/slowstack-24             921    1341206 ns/op  2797236 B/op     1846 allocs/op
BenchmarkPing/depth=625/faststack-24            3972     264099 ns/op    44901 B/op      651 allocs/op
BenchmarkPing/depth=625/slowstack-24             100   10104571 ns/op 13171325 B/op     8863 allocs/op
BenchmarkPing/depth=3125/faststack-24           4537     265868 ns/op    44900 B/op      651 allocs/op
BenchmarkPing/depth=3125/slowstack-24              8  140203673 ns/op 64903352 B/op    43949 allocs/op

We note the following:

- `faststack` is always faster than `slowstack`, but not by as much as we might expect
- `faststack` always uses  _significantly_ less memory.
- faststack's resource usage stops  increasing after depth 64, as we'd expect (since we hard-limited the callstack).
- slowstack's memory usage increases linearly with the depth of the stack, but it's runtime increases super-linearly as the stack gets larger. Why? I'm not sure, but my guess is that it's GC or fighting with the OS over the number of syscalls.

On a HDD, the difference is exaggerated,  but _neither_ implementation performs that well: 47 miliseconds for even the most trivial implementation. Even a 25-depth stack would produce a noticable pause (>180ms) from both `faststack` and `slowstack`.

goarch: amd64
pkg: gitlab.com/efronlicht/gin-ex
cpu: AMD Ryzen 9 5900 12-Core Processor
BenchmarkPing/depth=1/faststack-24              2294     476571 ns/op    14329 B/op      100 allocs/op
BenchmarkPing/depth=1/slowstack-24              2394     509789 ns/op   223090 B/op      105 allocs/op
BenchmarkPing/depth=5/faststack-24              1602     710711 ns/op    18492 B/op      140 allocs/op
BenchmarkPing/depth=5/slowstack-24              1381     813649 ns/op   306680 B/op      161 allocs/op
BenchmarkPing/depth=25/faststack-24              571    1878803 ns/op    39315 B/op      340 allocs/op
BenchmarkPing/depth=25/slowstack-24              500    2340672 ns/op   730874 B/op      444 allocs/op
BenchmarkPing/depth=125/faststack-24             324    3701402 ns/op    80024 B/op      653 allocs/op
BenchmarkPing/depth=125/slowstack-24             100   10164412 ns/op  2845715 B/op     1850 allocs/op
BenchmarkPing/depth=625/faststack-24             309    3739220 ns/op    80048 B/op      653 allocs/op
BenchmarkPing/depth=625/slowstack-24              22   51105918 ns/op 13394559 B/op     8870 allocs/op
BenchmarkPing/depth=3125/faststack-24            316    3756325 ns/op    80027 B/op      653 allocs/op
BenchmarkPing/depth=3125/slowstack-24              3  334763167 ns/op 66048874 B/op    43968 allocs/op

OK, what if we _can't_ find the source, like if we compiled with -trimpath?
c## Is this worth it?

There's two ways to frame this question: First, Is it worth optimizing a panic recovery handler? If so, should we be diving into the source code at all?

### Is it worth optimizing a panic recovery handler?

Everything comes at a cost. In this case, `faststack` cuts down on the memory usage and clock time significantly by using a more efficient algorithm and limiting the total stack depth, at the cost of doubling the length of the code and increasing it's complexity. `slowstack` was trivial to understand, and `faststack` is definitely not; it requires a new data structure, global variables, lazy initialization, it's _own_ panic recovery handler, and two separate sorts.

Why do we have a panic recovery handler in the first place?  To provide continuous service, even in the presence of software bugs. That is, a panic recovery handler is supposed to be a last-ditch protection against a bug that never should have made it into production.
These panics are _supposed_ to be rare; prevented by a suite of tests, CI, etc. In practice, well, all software is buggy, and Go programs can panic a lot. When I worked at an ISP, I had production bugs that triggered 20K recovery handlers, per minute, per server (on ~5 servers). If those stacks have 20 frames each, on average, and the files those frames came from average 20K, _8GiB_ of allocations a minute per server, or 40GiB of allocations per minute: - that's enough to bring most containers to a crawl, and even a relatively beefy modern PC might struggle to clean up all that garbage.

 While `faststack` cuts down on the memory usage and clock time significantly by using a more efficient algorithm and limiting the total stack depth, this comes at a cost: it doubled the length of the code and significantly increased it's complexity. `slowstack` was trivial to understand, but `faststack` is definitely not; it requires a new data structure, global variables, lazy initialization, it's _own_ panic recovery handler, and two separate sorts.

More worryingly, there's two performance problems that _faststack_ can't help with: first, reading from disk is a fundamentally slow operation; disks are slower than memory, and syscalls, such as the ones to read files, are expensive.  If the PC is reading from a HDD, both `faststack` and `slowstack` are actively slow. Secondly, most operating systems place a limit on the number of file handles a single process can open (`1024` by default on linux, iirc). Concurrent calls to `faststack()` might quickly eat up this limit. While these conditions are rare, they're likely to occur for at least some of Gin's users, who may struggle to diagnose the problem.

There are technical solutions for _those_ problems: for example, you could cut down on IO by havin a concurrency-safe cache for common file:line couples, and you could limit the total memory usage of _that_ by having it be a LRU cache, but that's adding _even more complexity_.

At some point, it's important to take a step back and ask yourself _whether or not all this work is worth it in the first place_.

### Is having a single line of source code annotation really helpful?

How much benefit are we really getting from this implementation? How likely is a single line of source code to help us to fix a bug, if we already have the **file**, **line**, and **function name**? Generally speaking, we'll need more context than a single line to diagnose the bug: we'll have to look at the source itself, not just a pinhole window into it. While it's a cool trick, this source code annotation is just that: a neat parlor trick, not a particularly helpful debugging tool. And there's one corner case where it could be really damaging: if the source code present on the host system differs from the version used to compile the executable, the source code annotations will be _wrong_.

Honestly, I think the Gin project would be better server by just using `runtime.Stack()` and forgetting about source code annotation. Sometimes simpler is better.
