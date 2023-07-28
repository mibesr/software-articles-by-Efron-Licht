THIS IS UNFINISHED AND UNPUBLISHED.
READ AT YOUR OWN RISK.

# test fast

A software article by Efron Licht\
July 2023

Part 1 of a series on building fast, resilient software, fast.

- [start fast: a guide to booting go programs quickly](https://eblog.fly.dev/startfast.html)
- [docker should be fast, not slow: a practical guide to building fast, small docker images](https://eblog.fly.dev/fastdocker.html)
- [have you tried turning it on and off again?](https://eblog.fly.dev/onoff.html)
- [test fast](https://eblog.fly.dev/testfast.html)

# Intro

The days of cowboy coding where ideas went straight to production are fading into the rearview mirror. Your average software project contains hundreds to thousands of tests that run before each and every deploy.

## tests should be

- fast. FAST! Only the rarest and most important tests should take longer than a millisecond to run. If a test _does_ take longer than that, it must be both
  - important
  - run in parallel with other tests!
- Parallelizable.
- Deterministic. Flaky tests (that is, tests that sometimes fail and sometimes succeed) are a cancer on a test suite. They erode trust. They lead to developers ignoring failures, and they erode your hard-earned speed gains by having developers retry the test suite over and over on doomed PRs because a test _might_ just be flaky.

## slow but important tests

Despite our best efforts, some tests are slow, taking hundreds of milliseconds or even seconds to run. We can mitigate the expense of these tests with these techniques.

- Run them in parallel, either within the test process or in separate processes using multiple machines.
- Start them as early as possible so they can 'catch up' with the rest of the test/deploy process.
- Find ways to cut them down into smaller sub-processes, perhaps by caching results from previous runs. This can be hard to explain, so let's go into a bit more detail
- This can be dangerous, though, as it can lead to your tests getting out of sync with your code. Use with caution.
- Run tests probabilistically. Let's suppose we're a big company with big scale. We have 1000 tests in the 'slow' category that take on average 1s each (after accounting for parallelism, etc), leading to a 15-minute integration test stage. Since we _are_ a big company, though, we have 100 PRs a day. If we test a _random subset_ of those tests on each run, we will probabilistically get full coverage rather quickly.

## just use maps for table-driven tests

Go programs often use tables to describe their test cases. I like this a lot.

## quickly initializing tests

Test binaries are just programs, and tests are just functions.  From a developer's point of view, though, tests run _more_ often than the main program, so initialization speed is even more important. The fact that test binaries are run in parallel means that commonly-imported packages have to initialize many times for a 'single' call to `go test ./...`. This can compound the pain of initialization time. Mostly, the techniques for speeding up how fast tests boot are the same as those for ordinary programs, a [topic I covered in detail previously](https://eblog.fly.dev/startfast.html), but to summarize:

- measure your initialization time to see if and where you have a problem (`GODEBUG=inittrace=1` still works here)
- use traditional optimization techniques to shave time off hot-spots
- use nonblocking eager intitialization or lazy initialization to free the main thread during startup (in tests, the 'main thread' is TestMain() for each package )
- store assets in a way that makes them easy to load quickly, or better yet, omit them entirely

Let's examine that last point in more detail. Many programs require a variety of assets to run (images, sounds, etc), which may either be loaded from disk or embedded in the binary. In either case, loading and processing these assets can take a long time, and this process will have to be repeated for every package that imports the assets package. But your 'short' tests shouldn't need these at all. Consider skipping loading them entirely, or falling back to a default asset. We can check the `testing.Short()` flag during normal program execution, but we have to make sure to parse the test flags too. Let's peek inside my game, **Tactical Tapir**'s source code for a demonstration of this technique:

```go
// in static.go

// populate the static maps in parallel
func init() {
 // if we're in a -short test, don't load the static assets to save time
 // a better idea might be to load a single test asset of each kind to substitute for everything
 testing.Init() // add the test flags to the CLI flag set
 flag.Parse()  // parse the flags to set testing.Short()
 if testing.Short() { 
  // it's a test AND -short is set
  log.Printf("-short: skipping static init")
  return
 }
 start := time.Now()
 wg := new(sync.WaitGroup) 
 wg.Add(4)
 go initMap(wg, Audio, "audio", loadAudio)
 go initMap(wg, Fonts, "font", loadFonts)
 go initMap(wg, Shader, "shader", loadShader)
 go initMap(wg, Img, "img", loadImg)
 wg.Wait()
 
 log.Printf("%-8s loaded %d files in %s", "all", len(Audio)+len(Img)+len(Fonts)+len(Shader), time.Since(start))
}
```

You can also avoid this issue by avoiding import-time behavior entirely or using the lazy-load and eager-nonblocking techniques discussed in my previous article.

## go-specific issue

The `go test` command builds a new test binary for each package, and executes them in parallel where possible. However, tests _within_ a package are run serially by default. Opt-in using `t.Parallel()`: tests will be run serially _up to_ the first call to `t.Parallel()`, and then in parallel after that. Execution of a subtest will be serial with regards to its parent test unless both the parent and the subtest call `t.Parallel()`. In practice, this just means **you need to call t.Parallel() once per layer of test**.

Let's demonstrate:

```go
// TestMul runs serially: no other tests in this package will run while it is running
func TestMul(t *testing.T) {
    if 5*2 != 10 {
        t.Fatal("5*2 != 10")
    }
}

// TestAdd runs in parallel with TestSub, and it's subtests will run in parallel both with each other and TestSub.
func TestAdd(t *testing.T) {
    t.Parallel() // Add will run in parallel with other tests in this package from this point on

    for _, tt := range []struct{
        a, b, want int
    } {
        {2, 2, 4},
        {3, 3, 6},
        {-128, 128, 0},
    } {
        tt := tt // capture range variable: see https://github.com/golang/go/discussions/56010 for details
        t.Run(fmt.Sprintf("%d+%d=%d", tt.a, tt.b, tt.want), func(t *testing.T) {
            t.Parallel() // this subtest will run in parallel with other subtests of TestAdd
            got := tt.a + tt.b
            if got != tt.want {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
            }
        })
    }
}
// TestSub will run in parallel with other tests in this package,
// but only one of its subtests will run at a time
func TestSub(t *testing.T) {
    t.Parallel() 

    for _, tt := range []struct{
        a, b, want int
    } {
        {2, 2, 0},
        {2, -2, 4},
    } {
        t.Run(fmt.Sprintf("%d-%d=%d", tt.a, tt.b, tt.want), func(t *testing.T) {
            got := tt.a - tt.b
            if got != tt.want {
                t.Errorf("Add(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
            }
        })
    }

}
```

We run the tests:

IN

```
go test -v ./...
```

```

OUT:
=== RUN   TestWriteFile
--- PASS: TestWriteFile (0.00s)
=== RUN   TestMul
--- PASS: TestMul (0.00s)
=== RUN   TestAdd
=== PAUSE TestAdd
=== RUN   TestSub
=== PAUSE TestSub
=== CONT  TestAdd
=== RUN   TestAdd/2+2=4
=== PAUSE TestAdd/2+2=4
=== RUN   TestAdd/3+3=6
=== PAUSE TestAdd/3+3=6
=== CONT  TestSub
=== RUN   TestAdd/-128+128=0
=== RUN   TestSub/2-2=0
=== PAUSE TestAdd/-128+128=0
=== CONT  TestAdd/2+2=4
=== RUN   TestSub/2--2=4
=== CONT  TestAdd/3+3=6
--- PASS: TestSub (0.00s)
    --- PASS: TestSub/2-2=0 (0.00s)
    --- PASS: TestSub/2--2=4 (0.00s)
=== CONT  TestAdd/-128+128=0
--- PASS: TestAdd (0.00s)
    --- PASS: TestAdd/2+2=4 (0.00s)
    --- PASS: TestAdd/3+3=6 (0.00s)
    --- PASS: TestAdd/-128+128=0 (0.00s)
PASS
ok   gitlab.com/efronlicht/blog/articles/testfast (cached)
```

'PAUSE' is the surefire sign of a parallel test. We can use `grep` to see exactly which tests are running in parallel, and just as we expected, it's `TestAdd` and it's subtests, and `TestSub`, but NOT it's subtests.

```sh
go test -v ./... | grep PAUSE
```

OUT

```
=== PAUSE TestAdd
=== PAUSE TestSub
=== PAUSE TestAdd/2+2=4
=== PAUSE TestAdd/3+3=6
=== PAUSE TestAdd/-128+128=0
```

I strongly advise you to design all tests to be as parallel as possible by default. In short, they should inject their dependencies rather then relying on synchronization with outside state (like globals), timing (like sleeps), or I/O (network calls, reading and writing files, syscalls). Let's go over how we can avoid those things for fast, parallelizable, and reliable tests.

### I/O

Proper use of interfaces can help you avoid I/O in tests. For instance, many instances of `*os.File` and `net.Conn` can be replaced with another `io.Reader` or `io.Writer`. File systems can be replaced with `io/fs` and outside servers can be replaced using `net/http/httptest`.

Let's demonstrate with an example.

```go
// Open the file at path and return all lines that contain pattern.
func findLinesMatchingInFile(path, pattern string) ([]string, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }

    var matches []string
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        if strings.Contains(scanner.Text(), pattern) {
            matches = append(matches, scanner.Text())
        }
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return matches, nil
}
```

We'd like to test the behavior of this function. While it's certainly possible to create a lot of files and test it that way, there are some potential issues that hurt it's portability and reliability:

- What if you don't have permissions on this filesystem? Do the names work on windows/linux?
- what about macs and their case-insensitive filesystems?
- how long will it take to create the file?
- what if multiple tests try to create the same file at the same time? (unlikely, I know, but possible.)
- did you remember to close and remove the file when you were done? did you get the order right? what if one or more failed?
- will that poison the filesystem for other tests?

We may run into problems with permissions, etc that might make this test unreliable. And the purpose of testing this function is not to find out whether or not `os.Open` works.

```go

func findLinesMatching(r io.Reader, pattern string) ([]string, error) {
    var matches []string
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        if strings.Contains(scanner.Text(), pattern) {
            matches = append(matches, scanner.Text())
        }
    }
    if err := scanner.Err(); err != nil {
        return nil, err
    }
    return matches, nil
}
```

Then we can simply pass `strings.Reader`s to the function pre-populated with the data we want to test:

```go
func TestFindLinesMatching(t *testing.T) {
    t.Parallel()
    for _, tt := range []struct {
        input, pattern string
        want          []string
    } {
        "foo\nbar\nbaz\n", "foo", []string{"foo"},
        "foo\nbar\nbaz\n", "bar", []string{"bar"},
    } {
        tt := tt
        t.Run(tt.input, func(t *testing.T) {
            got, err := findLinesMatching(strings.NewReader(tt.input), tt.pattern)
            if err != nil {
                t.Fatalf("findLinesMatching(%q, %q) = %v", tt.input, tt.pattern, err)
            }
            if len(got) != len(tt.want) {
                t.Fatalf("findLinesMatching(%q, %q) = %v, want %v", tt.input, tt.pattern, got, tt.want)
            }
            for i := range got {
                if got[i] != tt.want[i] {
                    t.Fatalf("findLinesMatching(%q, %q) = %v, want %v", tt.input, tt.pattern, got, tt.want)
                }
            }
        })
    }
}
```


Readers may be tempted to extend this to mocking out dependencies with some kind of framework. I find this to be a _very bad idea_. Mocks take a lot of space, both within the code and in your head, and they are both fragile to maintain and of questionable value. If you want to see if a database call works, you _need to test the database_. The false sense of confidence mocks give is worse than useless.

### grouping tests

group tests by speed and 'dependency group'. The most common & useful grouping is extremely coarse: "short" and "long" tests.

**Short** tests run in <5ms, require no outside dependencies, and are fully parallelizable. Most tests should be short tests.
**Long** tests fail one or more of the above criteria.  

Skip a long test by calling `t.Skip()` after checking `testing.Short()`

```go
func TestWriteFile(t *testing.T) {
    if testing.Short() {
        t.Skipf("SKIP %s: touches filesystem", t.Name()) // t.Name() is the name of the current test: here, "TestWriteFile"
    }
    // ... test code goes here ...//
}
```

IN

```
go test -short -v ./...
```

OUT

```
=== RUN   TestWriteFile
    testfast_test.go:7: SKIP TestWriteFile: touches filesystem
--- SKIP: TestWriteFile (0.00s)
```

Make sure to call `t.Skip()` or `t.Skipf()` _before_ `t.Parallel()`: forgetting to do so can cause unintuitive behavior or panics. Even long tests should be as short and simple as possible: the longer your tests
take to run, the less often you can afford to run them.

### dependency management

#### choose dependencies with fast and reliable tests

Everything I've said so far about YOUR tests goes

#### run your dependencies tests

People do not do this and I find it insane. If you don't trust _your_ code without tests, why would you trust anyone else?

#### have as few dependencies as possible

This is becoming a bit of a refrain. Dependencies add complexity to your code, length to the compilation, and size to the binary. You need to run your _Dep

### timing

We often sleep in functions when we're waiting for some condition to be met. We may wait 100ms for a database to be available, for instance. This sets an artifical floor on the latency of your tests (that is, you make them **at least that slow**, when they could potentially be a dozen times faster).

In general, try to avoid sleeping entirely by using a channel, mutex, waitgroup, or condition variable to signal when the condition is met, roughly in that order of preference. That is, **push** synchronization events from your dependencies to the test setup code. If you can't do that (e.g, you're waiting for a database to be available), you can set up repeated polls at `1-2ms` intervals and then push updates from there. Waiting and guessing is by far the worst way to synchronize.

Tests are just programs, so the same advice for quick initialization applies as in traditional program.s

If you _must_ wait, do so _within_ the tests, _past_ the point where `t.Parallel()` is called. This will free Go's scheduler to run other tests while dependencies are being set up.

    ```go
    func main() {
        go setupRedis()
        go setupPostgres()
        go setupRabbitMQ()

        time.Sleep(1*time.Second) // wait for redis, postgres, and rabbitmq to be ready
        serveHTTP()
    }

```

To quickly review the math, for any given test, if we run it with probability `p`, then the probability of _not_ running it is `1-p`. If we run it `n` times, the probability of _never_ running it is `(1-p)^n`. If we want to be 99% sure we've run it at least once, we can solve for `n`:  `(1-p)^n = 0.01`, or `n = ln(0.01) / log(1-p)` To expand that out to any target probability, we can use `n = ln(1 - target) / log(1 - p)`.

Let's fill out a table of `n` for various values of `p` and `target`:

```go
https://go.dev/play/p/mu1Grq9z5FS

func main() {
 const format = "|%v|%v|%v|\n"
 fmt.Printf(format, "% chance/run", "desired coverage", "min runs needed")
 fmt.Printf(format, "---", "---", "---")
 for _, p := range []float64{0.05, 0.1, 0.2, 0.3} {
  for _, tgt := range []float64{0.99, 0.999, 0.9999} {
   const format = "|%0.2f|%0.4f|%v|\n"
            runs := int(math.Ceil(math.Log(1-tgt)/math.Log(1-p))))
   fmt.Printf(format, p, tgt, runs)
  }
 }
}
```

|% chance/run|desired coverage|min runs needed|
|---|---|---|
|0.05|0.9900|90|
|0.05|0.9990|135|
|0.05|0.9999|180|
|0.10|0.9900|44|
|0.10|0.9990|66|
|0.10|0.9999|88|
|0.20|0.9900|21|
|0.20|0.9990|31|
|0.20|0.9999|42|
|0.30|0.9900|13|
|0.30|0.9990|20|
|0.30|0.9999|26|

If our deploys run frequently enough, we'll quickly asymptote to full coverage. Even at `p=0.05`, we can be reasonably certain every single test has run _at least_ every 3 days if we run 100 PRs a day. This technique is a compromise for extremely large-scale codebases, not a goal to strive for.

For one thing, management is usually not happy with 'probabilistic' coverage, and once you _have_ discovered a failure, there's a significant infrastructure burden necessary to find the exact PR that caused the failure: you'll need relatively advanced automation to narrow down the set of PRs (using bisection, generally) that could have caused the failure and run the failing test(s) on each of them. It's generally, much simpler to have a suite of quick tests you can run every single time. But if you find yourself having to make the choice between cutting tests or slowing down deploys, probablistic coverage can serve as a reasonable compromise. ..A good compromise could be using probablistic coverage for individual PRs as they stack up, but running the full test suite both on a regular schedule (say, a few times a day) and before deploys to production. Even so, with large numbers of deploys, you'll end up saving hundreds of hours of developer & machine time.

## start long-running tests early in the deploy process

Don't run all your tests in sequence. Start each stage of testing as soon as you possibly can, and avoid blocking on the results of previous stages except where necessary. Pre-production environments exist for a reason: it's OK to allow a deploy to proceed to a staging environment even if some tests are still running, so long as you protect production.

Run as many steps concurrently as possible. If you find yourself 'stepping' on a staging environment, rather than slowing down the deploy process, consider adding another staging environment as a 'buffer' layer

Generally, we'll have some very fast tests we can use as a quick-and-dirty check to see whether our code is any good at all. Then we'll have some longer-running tests that are more thorough and might need to touch external resources like DBs or file systems. Finally, we'll have some end-to-end tests that mimic or duplicate real user behavior in a production-like environment.

Here's a diagram of a typical test pipeline: (see the original MERMAID DIAGRAM HERE `<INSERT THAT LINK>`)

<TODO: rebuild this part.>
graph LR
subgraph unit_test
fmb
[format/build/lint] --> ft{fast/unit test}
ft --> st[slow/integration tests]
end
subgraph integration_test
end
subgraph staging

end
subgraph prod
end
end

```

![diagram](https://mermaid.ink/img/eyJjb2RlIjoiZ3JhcGggTFJcblxuc3ViZ3JhcGggQ0k6XG5mYmx7Zm9ybWF0L2J1aWxkL2xpbnR9IC0tIHBhc3MgLS0-IGZ0e2Zhc3QvdW5pdCB0ZXN0c31cbmZibCAtLT4gc3RcbnN0e3Nsb3cvaW50ZWdyYXRpb24gdGVzdHN9XG5lbmRcbnN1YmdyYXBoIHN0YWdpbmdcbmZ0IC0tLT4gZGRbZGVwbG95IGRldl1cbmRkIC0tPiBlMmV7ZW5kIHRvIGVuZCB0ZXN0c31cbmVuZFxuc3ViZ3JhcGggcHJvZFxuZTJlIC0tPiBkZXBsb3lfcHJvZFxuc3QgLS0-IGRlcGxveV9wcm9kXG5lbmRcbiIsIm1lcm1haWQiOnsidGhlbWUiOiJkZWZhdWx0In0sInVwZGF0ZUVkaXRvciI6ZmFsc2V9)

And what it might look like in a .gitlabci.yml file (with all the little details hidden in some `Makefile`).


```mermaid

```yaml
stages:
  - unit_test
  - integration_test
  - staging
  - prod
unit_test:
    stage: unit_test
    script:
        - go test -short ./...
integration_test:
    needs: ["unit_test"]
    stage: integration_test
    script:
        - make integration_test
        - go test -v ./...
deploy_staging:
    needs: ["unit_test"]
    stage: staging
    script:
        - make deploy_staging
end_to_end_test:
    needs: ["integration_test", "deploy_staging"]
    stage: staging
    script:
        - make end_to_end_test
deploy_prod:
    needs: ["integration_test", "end_to_end_test"]
    stage: prod
    script:
        - make deploy_prod
```

## avoid repeating work

## Flaky tests: public enemy #1

There is nothing quite so damaging to test infrastructure as a flaky test: a test that _sometimes_ passes and sometimes fails. They are quite literally worse than useless:

- they take time to run
- they tell you nothing about the correctness (or lack thereof) of your code
- they erode trust in the test suite
- developers get used to ignoring failures or simply retrying a failing test suite, meaning a suite is not treated as truly failed until multiple runs have failed.

## dependency resolution

There are three main strategies for dependencies in tests.
"Mock" - replace the dependency with a fake implementation that returns canned responses. **In general**, mocks are a bad idea.

Instead of having a function like this:
