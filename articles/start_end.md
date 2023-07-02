# quick-start quickstart

a bullet-point guide to making your program start faster

## what?

computers today are fast, but programs start slowly. this is bad.

## are you sure?

- `<TODO: insert twitter post by casey muratori or something showing a windows XP computer booting visual studio way faster than a computer with M.2 SSDs and 32GB of RAM booting VSCode>`

this is wasteful of time, money, & electricity. it makes programs actively unpleasant to use. we should have the dignity to care. take pride in your work! your program may not do anything, but by god it should do nothing _fast_.

### user experience

- people will give up on your program if it takes too long to start.

> **dev:**
> i wanted to use this neat tool, but it took too long to start / I couldn't figure out how to use it
> **user**: i keep clicking the 'give you money' button but it's not loading. guess I won't.

### program kinds that MUST start quickly

- interpreters
- command-line tools meant to be pipelined
  java is a tremendously popular language, but dominated by python, go, rust, etc for command-line tools because it's so slow to start: not surprising for java programs to take 10s of seconds to start, even for trivial programs
- 'serverless' cloud functions & event handlers of all kinds

## core of speed

only 3 ways to increase speed:

- do less work (DRY)
- do that work faster (optimization)
- do multiple things at once (parallelization)

## what's "starting" a program?

'starting' a program can have a variety of definitions. what is 'starting' a program to you? we generally think of a program as 'started' when it's ready to accept input from the user(s), but before you even get there, you may need to do one or more of the following:

- running a binary/script
- that you compiled
- via a shell script that sets up the environment
- that dynamically links to a bunch of OS-specific libraries
- after downloading a bunch of dependencies
- and compiling them
- and setting up and running a database
- in a container
- in a vm
- provisioned by a cloud provider
- in a kubernetes cluster
- and again in staging...
- and again in production...
- in two different regions...

we will split this into two parts:

- running a binary
- those other things

## running a binary

assuming your program starts, how do we get from `./myprogram` to `press any key to begin` or `serving HTTP on :8080` as fast as we can?

### choice of language and tools matters

#### programming language

- interpreted languages are slow to start & getting slower
  - exceptions: Lua, sh/bash
- compiled languages usually slow to build, fast to start
  - except Go, which is fast to build and fast to start
  - and Java, which is slow to build AND slow to start
- Compile time hall of shame: Java, C++, Rust, Haskell

#### program initialization

programs work vaguely like this:

- load the program into memory
- initialize imports (e.g. `init()` in go, or anything outside a function in python). **this is the first sloslow part.**

### measuring

- first, get a general sense of initialization time. what's your overall boot time? that is
  - let's say we start a program at`t0`. we hit the first line of `main` at `t1`, and we begin serving user input at t2. let's call `t1-t0` "import time", and `t2-t0` "start_time", and `t2-t1` "init time". That is, `start_time = import_time + init_time`.
  - Track `t0` outside, and `t1` and `t2` inside your program. You can pass `t0` in as an envvar, or it may be part of your language's runtime.

    A golang example:

  ```go
    func main() {
        t1 := time.Now()
        /* setup your program here */
        t2 := time.Now()
        if t0, t0err := time.Parse(time.RFC3339, os.Getenv("BOOT_TIME")); t0err == nil {
            fmt.Printf("import time: %-4.2f\n", t1.Sub(t0).Milliseconds())
            fmt.Printf("start time: %-4.2f\n", t2.Sub(t0).Milliseconds())
        }
        fmt.Printf("init time: %-4.2f\n", t2.Sub(t0).Milliseconds())
        
    }
    ```

### improving pre-main init time

- program start time (e.g. `time ./myprogram`)
  in Go, you can use `GODEBUG=inittrace=1` to trace initialization. this will give you a list of packages and how long they took to initialize.
  
  ```

- run main() or equivalent
 languages allow arbitrary import-time code to run. this is often slow. keep track of what your program is doing at startup. simple way to do so in go:

    ```go
    package traceinit
    import "time"
    var prevFile, prevLine = "", 0
    var last = time.Now()
    // Trace prints the time since the last call to Trace().
    // Call this in every package's init() function to see how long each package takes to initialize.
    func Trace() {
        now := time.Now()
        elapsed := now.Sub(last)
        last = now
        // sidenote: many IDEs recognize file:line or file:line:col as a link, making it easy to jump to a misbehaving package.
        _, file, line, _ := runtime.Caller(1) // 1 = caller of Trace()
        fmt.Fprintf(os.Stderr, "init %s:%-4d: %4.2fms since %s:%-4d\n", file, line, elapsed.Seconds()*1000, prevFile, prevLine)
    }
    ```

    This will give limited output, since it d

  Example output:
  ```

computers are getting faster and faster, but programs are getting slower and slower.  this is gross. it's wasteful of:

- time
- money
- electricity.
it makes programs actively unpleasant to use.
    A similar technique can be used in most languages with access to runtime information or stack traces. E.G, in python, you can use `__file__` or `inspect`.

#### "code" dependencies (libraries, frameworks, etc.)

have as few as possible.

- downloading & building deps takes time.
- downloading & building _their_ deps takes time.
- reading a library is often better than using it. learning is a zero-cost abstraction.
- not uncommon to see >=3K lines of dependencies for a single function. don't do this.
- linking is surprisingly slow. prefer static linking where possible.
-

- #### biggest dependencies of all are external programs

- your program only boots as fast as the slowest program it depends on.
if you just need a function,
- `node`, `yarn`, `pip`, `gem` are all pretty bad
- `go` is pretty good.
- `cargo` (rust) is pretty good, but initializing the `crates.io` index is _extremely_ slow. make sure you don't have to do this every time you build.
- `brew` is rather slow but not awful
- `apt`, `apk`, `pacman` are all OK
Most dependency management tools have

### but the **number** of languages matters more

- build-time hell comes from thousands of overlapping layers of YAML, shell scripts, and Makefiles, each of which is a different language with different semantics and different tooling. especially prevalent in backend, 'cloud' or 'microservice' development, where you have kube, docker, terraform, helm, kafka, etc etc etc all for one 'simple' service.
- even seasoned developers can't be masters of _everything_
- easy to enter the 'pit of despair' w/ unfamiliar configuration languages where its hard to tell what if anything is taking the time
- if you already have `brew`, _just_ use `brew`. choose one.
- if you have `make`, _just_ use `make`. (aside: c++ devs may have no choice but to use dozens of build tools. i'm sorry.)
- resist the urge to add a new tool or language to solve a problem.
- try writing a simple script first before reaching for a new tool. it's often faster and easier than you think.

### avoid repeated work

- the following steps should happen `0..1` times per build:
  - downloading dependencies (e.g. `go mod download`)
  - copying source code to build a container
  - compiling test binaries
  - compiling the program
  - building artifacts (documentation, docker images, binaries for different platforms, etc)
  - running unit tests
- the following steps should happen `1..n` times per build, where `n` is the number of environments you're testing in:
  - running integration tests
  - running end-to-end tests
  - running benchmarks

### docker-specific tips

#### docker: minimize cache invalidation

Docker caches layers in the order they're written in the Dockerfile. Once you invalidate the cache, all subsequent layers are invalidated.

- Start your dockerfile as late in the process as possible.
- Things that will 'NEVER' change (e.g, OS, compiler) should be in the parent image, not the `Dockerfile`.
- Order your `Dockerfile` from LEAST likely to change (TOP) to MOST likely to change (BOTTOM).
- If you deploy multiple times for a single version, separate build from run, so only one machine has to download GCC or the Go compiler, (third example of big dependency)

#### docker: smaller is better. faster to download, faster to upload, faster & cheaper to run and store

- small starts with the base image. prefer `alpine` to `ubuntu`, `slim` versions of debian, etc.

    (all images are linux/amd64, as of 2023-06-28)
    |image|tag|size (compressed, MiB)|
    |---|---|---|
    |`alpine`| 3.18.2 | 3.24 |
    |`golang` | 1.20.5 | 314.84|
    |`golang`|1.20.5-alpine3.18| 99.77|
    |`python` | 3.11.4-slim-bullseye | 45.62 |
    |`python`| latest | 360.43|

- install only the specific tools you need. e.g, `apk add --no-cache git` instead of `apk add --no-cache git curl wget tar gzip`
- avoid extraneous dependencies. think of the dockerfile as the _minimum_ set of dependencies needed to build & run your program.
- each layer of a dockerfile is
- separating the build and run images can cut down on unnecessary downloads & copies.
  - ideal docker run image:

  ```docker
  FROM alpine:<someversion> # don't use latest, pin a version
  COPY app /app --from=builder # copy the binary from the builder image
  ENTRYPOINT ["/app"] # run that binary
  ```

- remove build artifacts or extraneous files and dependencies before building the final image
- separate BUILD and RUN images can help here

#### docker: minimize cache invalidation

Docker caches layers in the order they're written in the Dockerfile. Once you invalidate the cache, all subsequent layers are invalidated.

- Start your dockerfile as late in the process as possible.
- Things that will 'NEVER' change (e.g, OS, compiler) should be in the parent image, not the `Dockerfile`.
- Order your `Dockerfile` from LEAST likely to change (TOP) to MOST likely to change (BOTTOM).

#### docker:copy

COPY-ing files invalidates the cache if any files change.

- `COPY` as few files as possible.
- Prefer granular `COPY` like

    ```docker
    COPY go.mod go.sum ./ # only busts the cache when go.mod or go.sum change
    RUN go mod download # can be cached
    COPY src ./src # only busts the cache when src changes
    ```

    to 'broad' `COPY` like

    ```docker
    COPY . . # too broad, busts the cache when ANYTHING in root changes
    RUN go mod download 
    RUN go build
    ```

    then any time you change a line in your source code, docker will have to re-download all your dependencies. instead, something like this:

will allow docker to cache the `go mod download` step, and only invalidate the cache when you change your dependencies.

-

### git-specific tips

- #### a small repo is a fast repo

  - `git-gc` can trim 70% or more off repos.
  - avoid large files.
    - store them w/ tools like `git-lfs` can help. both `github` and `gitlab` have decent artifact storage solutions.
  -

- #### recurse-submodules can be done in parallel

    Try to avoid submodules, but if you must, at least do it in parallel:

    ```sh
    git fetch --recurse-submodules --jobs=8 # download objects and refs from a repo & its submodules, up to 8 at a time
    git submodule update --init --recursive --jobs=8 #  Install a repository's specified submodules, 8 at a time
    ```

- clone just the branch you need with

    ```sh
    git clone --single-branch --branch <branchname> <remote-repo> #  clone a specific branch and nothing else
    git clone --depth 1 <remote-repo> # clone only the most recent commit of the default branch
    ```

- clone a subset of files or directories with [`git sparse-checkout`](https://git-scm.com/docs/git-sparse-checkout) (this is usually more trouble than it's worth, but can handy for mono-repos)

## questions to ask yourself

### how do fellow developers usually interact with your program? how long does that take?

- "first time" interaction
  - how long does it take to get the program running?
  - how long does it take to get the program running _correctly_?
  - how long does it take to get the program running _correctly_ _on a new machine_? does it clearly guide you to the right place to get started? have you tried pointing someone else at your README and seeing if they can get it running without help?
- "usual" interaction
  - how long from making a change to seeing that change in action?
    - clone time
      - how long does it take to clone the repo?
      - how big is the repo?
    - solutions: shallow clones, sparse checkouts, [git filter-repo](https://github.com/newren/git-filter-repo/)
    - compile time
      - downloading dependencies is always slow!
        - slow: maven, npm, etc
        - slow-ish: cargo, go.
      - solution: vendor dependencies! build caches are hard to get right, vendoring is easy.

      -
    - deploy time
    - test time
    - boot time
      - how long does the program take to start? what's the 'cold boot' time? what's the 'warm boot' time?
      - how long does it take to get to a point where you can interact normally with the program?
      - what's taking the time?
        - is it the language runtime? java is horrible w/ this.
        - is it imports, import-time behavior? python is horrible w/ this.
        - are you establishing connections to databases, etc that you might not need? are you doing heavyweight constructors, etc?

### strategies for boot-time improvement

- how do users usually interact with your program? how long does that take?
-
-

how long do these things take? how reliable are they?
