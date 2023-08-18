THIS IS UNFINISHED AND UNPUBLISHED.
READ AT YOUR OWN RISK.

# test fast

A software article by Efron Licht\
July 2023

- [start fast: a guide to booting go programs quickly](https://eblog.fly.dev/startfast.html)
- [docker should be fast, not slow: a practical guide to building fast, small docker images](https://eblog.fly.dev/fastdocker.html)
- [have you tried turning it on and off again?](https://eblog.fly.dev/onoff.html)
- [test fast](https://eblog.fly.dev/testfast.html)

# Intro: What's wrong with my tests, anyway?

Programming work varies wildly in scope. Some programming work is the exciting kind where you come up with clever algorithms, pack bits, or implement clever new features. But just as much programming work is 'routine': fixing spelling errors, updating configuration, adding an extra URL parameter to a HTTP request. The software engineering 


Your average software project contains hundreds to thousands of tests that run before each and every deploy. While this has a number of important benefits, such as


- finding bugs before they hit production
- guarding against potential regressions introduced by new features or other bugfixes
- acting as a "living" set of documentation and examples for the codebase
- giving a nice warm fuzzy feeling to engineers & managers,
  
**Tests are not a universal panacea: in fact, slow or unreliable tests can cause more damage than they prevent.** A healthy test suite is about far more than coverage: useful tests are _fast_, _reliable_, and _deterministic_. In this article, we'll cover _why_ speed & consistency so crucial to a healthy test suite, and then go over _how_ to achieve them in practice.

This article is a loose sequel to some of my other posts on infrastructure and speed. I'll link to them where appropriate, but you don't need to read them to understand this article.

## why do speed & determinism matter so much?

Programming work varies wildly in scope. Some programming work is the exciting kind where you come up with clever algorithms, pack bits, or implement clever new features. But just as much programming work is 'routine': fixing spelling errors, updating configuration, adding an extra URL parameter to a HTTP request. We build automated test suites to give ourselves a confidence that when we make changes like this that we haven't broken anything.

 but many bugs are incredibly trivial: spelling an environment variable as `DBNAME` instead of `DB_NAME`, checking an error, adding an extra URL parameter to a HTTP request - the kind of bug that _should_ take thirty seconds to fix. Much of the most valuable programming work consists of quick iteration on problems of this kind as they shape the codebase into something useful. Only one problem: for the vast majority of professional codebases, the software test suite takes ten, thirty, or sixty minutes to run. By the time the test suite finishes, the programmer has been called into a meeting, pulled away by another bug, been out to lunch, or simply lost their train of thought. Either way, what should have been a 'quick fix' drags into the next day, or week, or month. **_Slow tests are an enormous productivity killer, a hidden demon sabotaging every attempt to build quality software._**

This problem gets so much worse when flaky tests enter the picture. There is nothing quite so damaging to test infrastructure as a flaky test: a test that _sometimes_ passes and sometimes fails. Software Engineers universally develop a defense mechanism against flaky tests: they ignore them. Now, remember, **the entire purpose of a test suite is to tell you that your code needs to be fixed**. But since flaky tests are not a reliable indicator, developers ignore them, and they are training themselves and others to ignore the test suite, making the entire suite less and less reliable. Over time, as the flaky tests build up, developers assume _all_  failures are flaky, and they start simply ignoring _all_ failures and over-riding the safeguards that prevent broken code from being deployed. This concept, called "alarm fatigue", is well known in a variety of other fields. Disasters from [plane crashes](http://www.nytimes.com/1997/08/08/world/pilot-error-is-suspected-in-crash-on-guam.html?scp=8&sq=Korean%20Guam&st=cse) to  [exploding oil rigs](http://www.nytimes.com/2010/07/24/us/24hearings.html?scp=1&sq=Deepwater%20alarm&st=cse) to the famous meltdown at the chernobyl nuclear power plant were all, in part, due to alarm fatigue.

It's hard to overstate how demoralizing this can be to a Software Engineer who is bursting  w/ good ideas about where and how to fix the problem. Some of the worst nights of programmers' lives involve convincing themselves they'll stick around "just until they fix this simple bug", only to find themselves still at the office at 1am, exhausted, waiting for yet another agoniizingly slow test run to finish. The product managers have to explain to the execs how a seemingly 'trivial' bug took days or weeks to fix. The exhausted progammers are then berated for their lack of productivity, or worse yet, their lack of 'ownership' of the problem, when they've run themselves ragged trying to fix it. This tendency to blame [individuals rather than systems](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC1117770/pdf/768.pdf) for problems is well-known in organizational psychology. But because **infrastructure is all-too-often effictively invisible to leadership, it rarely gets prioritized**, even when it's the root cause of what leadership sees as a 'productivity' problem. Often times, the leadership is too far away to see the problem, and the junior engineers who are most affected by it are too 'close' to the problem to be able to really see it. They know something's wrong - everything takes too
long, everything is too hard - but they don't know _why_ - and even if they do, they have no idea how to fix it.

And in an attempt to regain the lost speed and regain the trust of leadership & product management, programmers will start cutting  _more_ corners: ignoring _more_ tests, and over-riding _more_ safeguards. It's a death spiral through good intentions.

## Good tests

OK, so if that's what BAD tests are like, what are good tests like?
**Ideally**, Good tests are:

- 
- Start fast. Tests should have little or no 'initialization lag' before they start running. This means they should avoid I/O, network calls, and other slow operations, and they should avoid setting up dependencies, allocating memory, or other expensive operations unless absolutely necessary.
- 
- Fail fast.
- Fast. FAST! Only the rarest and most important tests should take longer than a millisecond to run. If a test _does_ take longer than that, it must be both
  - important
  - run in parallel with other tests!
- Parallelizable.
- Deterministic. They should always pass or always fail, **and they should do so for the same reason every time**.
- Have as few dependencies as possible.


## test parti
## fast

Test binaries are just programs, and tests are just functions. To a certain extent, you make tests fast the same way you make any other program fast: by use of appropriate data structures, avoiding I/O and allocation, and so on. However, a test has an unusual structure for a program: it usually spends most of it's time initializing, and very little time actually running. From a developer's point of view, though, tests run _more_ often than the main program, so initialization speed is even more important.

Go, for instance, builds multiple test binaries, one for each package with tests, and runs them in parallel. This means that _each test binary_ has to initialize all the packages it imports, and then the test binary itself has to initialize. This means that a single package may have to be initialized dozens or hundreds of times in a single `go test ./...` invocation, compounding the pain of initialization time. Mostly, the techniques for speeding up how fast tests boot are the same as those for ordinary programs, a [topic I covered in detail previously](https://eblog.fly.dev/startfast.html), but to summarize:

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

## slow but important tests

Despite our best efforts, some tests are slow, taking hundreds of milliseconds or even seconds to run. We can mitigate the expense of these tests with these techniques.

- Run them in parallel, either within the test process or in separate processes using multiple machines.
- Start them as early as possible so they can 'catch up' with the rest of the test/deploy process.
- Find ways to cut them down into smaller sub-processes, perhaps by caching results from previous runs. This can be hard to explain, so let's go into a bit more detail
- This can be dangerous, though, as it can lead to your tests getting out of sync with your code. Use with caution.
- Run tests probabilistically. Let's suppose we're a big company with big scale. We have 1000 tests in the 'slow' category that take on average 1s each (after accounting for parallelism, etc), leading to a 15-minute integration test stage. Since we _are_ a big company, though, we have 100 PRs a day. If we test a _random subset_ of those tests on each run, we will probabilistically get full coverage rather quickly.

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


### dependency management

#### choose dependencies with fast and reliable tests

Everything I've said so far about YOUR tests goes for other people's tests, too.

#### run your dependencies tests

People do not do this and I find it insane. If you don't trust _your_ code without tests, why would you trust anyone else? If their tests are slow, maybe don't run them _every time_, but at least run them a good thirty times every time you update a version to make sure _their_ tests are reliable.

#### have as few dependencies as possible

This is becoming a bit of a refrain. Dependencies add complexity to your code, length to the compilation, and size to the binary. They also add a lot of potential points of failure. If YOU write a piece of code, and it breaks, at least you have some idea how and why you wrote it.

### timing

We often sleep in functions when we're waiting for some condition to be met. We may wait 1s for a database to be available, for instance, a pattern that looks like this:

```go
func TestMain(m *testing.M) {
    // EXAMPLE ONLY: don't do this
    go func() {
        db, err = setupPostgres()
        if err != nil {
            log.Fatalf("postgres: %v", err)
        }
    }
    go func() {
        redis, err = setupRedis()
        if err != nil {
            log.Fatalf("redis: %v", err)
        }
    }
    time.Sleep(1 * time.Second) // wait for the database to be available
    m.Run()
}
```

This sets an artifical floor on the latency of your tests (that is, you make them **at least that slow**, when they could potentially be a dozen times faster).

In general, try to avoid sleeping entirely by using a channel, mutex, waitgroup, or condition variable to signal when the condition is met, roughly in that order of preference. That is, **push** synchronization events from your dependencies to the test setup code.

Sometimes the service you're depending on isn't nice enough to signal _you_ when it's ready and you need to poll it instead. If you absolutely must, set up repeated polls at 1-2ms and then push updates from there. Go's channels can be a good way to do this: send an error or nil for each dependency on a channel, and just drain one for each dependency you're waiting on.

The folowing example shows **one way**  to convert a poll into a push:

```go

package main_test

var redis *redis.Client
var db *sql.DB

func TestMain(m *testing.M) {
   // setup dependencies. when each dependency finishes or times out, send a message on the channel
   res := make(chan error, 2) // postgres & redis
   ctx, cancel := context.WithTimeout(context.Background(), 1 * time.Second)
   defer cancel()
   go func() {
      var err error
      db, err = setupPostgres()
      if err != nil {
         res <- fmt.Errorf("postgres: %w", err)
         return
      }
      for {
        switch err := db. 
      }


   }()
   go func() {
      var err error
      redis, err = setupRedis()
        if err != nil {
            log.Fatalf(" to redis: %v", err)
        }
        for { // keep pinging until we get a response or time out
            switch err := redis.PingContext(ctx);  {
                case err == nil:
                    res <- nil
                    return
                case errors.Is(err, context.DeadlineExceeded:)
                    res <- fmt.Errorf("redis: %w", err)
                    return
                default:
                    time.Sleep(2 * time.Millisecond)
            }
        }
   }
    // wait for all dependencies to be ready
    for i := 0; i < 2; i++ {
        if err := <-res; err != nil {
            log.Fatalf("failed to connect to %s: %v", err)
        }
    }
   os.Exit(m.Run())
}
```

Tests are just programs, so the same advice for quick initialization applies as in traditional program.s

If you _must_ wait, do so _within_ the tests, _past_ the point where `t.Parallel()` is called. This will free Go's scheduler to run other tests while dependencies are being set up.

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

## dependency resolution

There are three main strategies for dependencies in tests.
"Mock" - replace the dependency with a fake implementation that returns canned responses. **In general**, mocks are a bad idea.

"Injection" - pass the dependency in as a parameter to the function. This is the best option in general, but it can be a lot of work to refactor a codebase to use it.

Instead of having a function like this:
