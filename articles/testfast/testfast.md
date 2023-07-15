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
- Find ways to cut them down into smaller sub-processes, perhaps by caching results from previous runs. This can be dangerous, though, as it can lead to your tests getting out of sync with your code. Use with caution.
- Run tests probabilistically. Let's suppose we're a big company with big scale. We have 1000 tests in the 'slow' category that take on average 1s each (after accounting for parallelism, etc), leading to a 15-minute integration test stage. Since we _are_ a big company, though, we have 100 PRs a day. If we test a _random subset_ of those tests on each run, we will probabilistically get full coverage rather quickly.

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
