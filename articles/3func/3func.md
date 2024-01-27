# three views on functions


## outline
- introduction: what's a function anyways?
  - "function" is one of the most common words in programming
  - but we don't all agree on what a function is
  - a 'function' is actually many different things depending on context
  - understanding these different viewpoints can help us write better code
  - all of these viewpoints 'have a point', but they're not all equally useful (hint: i like the subroutine view)
- functions as subroutines
  - motivation: 
    - go code doesn't actually exist
    - computers are fundamentally equivalent: you have a tape head and an instruction pointer and one or more places to store data
    - variables and functions aren't real.
## 'gotwo': a restricted subset of go that will help us understand what a function is.

### 'gotwo' rules:
- only two function calls: `main` and `println`.
- `()` and `{}` may only be used for `main`, `if`, `print` and `println`.
- no multiple assignment.
- the only types are `int` and `bool`. bools are only valid in `if` statements.
- flow control: no `for`, `else`, `select`, or `range`. `if` must be limited to a single `goto`. (that is, no implicit jumps)
- declarations: no `:=` or `const`.
- how do we do flow control? `goto` and `if`.
- how do we do loops? `if` and `goto`.
- example program 1: `gotwo/fib.go`
- example program 2: `gotwo/prime.go`
- how to define a subroutine: `goto` and `if`. but how can we get back to where we were?
  - POINTER JUMPS: `goto` and `if` can only jump to a label: how do we jump to a specific line?
- extending "gotwo" with pointer jumps: `gotwo/pointers.go`
- this is how functions actually work: you jump to a label, and then you jump back.
- the SPECIFIC VARIABLES you store the return address, where to put the return value, and the arguments are called the ABI (application binary interface)
- now we can see what calling a function involves:
    - copy current arguments somewhere so they're not overwritten by the function (this is more complicated than we'll go into here - look into "stack frames" if you want to know more)
    - store argument(s) & return address according to ABI
    - jump to function
      -  function does its thing
      -  function stores return value according to ABI
   -  jump back to stored return address
   -  copy return value(s) to where they're needed
   -  continue execution
- this is how functions work in every language. details of the ABI vary across programming language, architecture, and operating system. [you can see go's ABI(s) here](https://go.googlesource.com/go/+/refs/heads/master/src/cmd/compile/abi-internal.md).
- functions **are subroutines with a specific ABI**.
  - exist to cut down on code duplication (easier to read, easier to write, can have performance benefits)
  - but you always\* pay for the cost of the function call. \* (compiler optimizations can sometimes remove the cost of a function call, but you shouldn't count on it).
  - subroutines are great, but not free.
  - this gives us insight into how a "function pointer" is implemented (that is, a variable like `var f func(int) int`): it's just an integer that stores the address of a subroutine.
  - at runtime, there's no need to keep track of what "kind" of function it is, since **all functions share the same ABI**.

- functions as mathematical transformations
this is the origin of the word "function" in programming. a function is a mathematical transformation.
function rules:
- has a range of inputs, called (domain). no inputs is a valid domain.
- has a range of outputs, called "codomain" or "range".
- **every input has exactly one output**.
- multiple inputs can have the same output.

Some notes on terminology: since we're talking about math instead of computers now, I'll use the term `real` for ordinary numbers like `1` or `3.1` or `π`.


## example functions:
`zero` maps any input to 0.
`next` maps any integer to the next integer (e.g. `next(1) = 2` and `next(2) = 3`).
`add` maps any two integers to their sum.
`ceil` maps any real number to the smallest integer greater than or equal to it.

| function | domain | codomain | example | in go |
| -------- | ------ | -------- | ------- | ----- |
| zero | any | int | zero() = 0 | return 0 |
| next | int | int | next(1) = 2 | return x + 1 |
| add | (int, int) | int | add(1, 2) = 3 | return x + y |
| ceil | real | int | ceil(1.5) = 2 | return int(x + 1) |
| floor | real | int | floor(1.5) = 1 | return int(x) |
| square | real | real | square(2) = 4 | return x * x |

Some **well-defined functions are impossible to compute.** The most famous is the halting problem: given a program and some input, will the program ever stop running? That is, suppose we have the set of all possible gotwo programs in one file under 4KiB as input, and we want to know if each program halts. This is a function from gotwo programs to bools. It's clearly well-defined: the answer is either yes or no.

| transformation | description | mathematical function | subroutine |
| -------------- | ----------- | ---------------------- | ---------- |
| `next` | maps any integer to the next integer | yes | yes |
| `add` | maps any two integers to their sum | yes | yes |
| `gamma` | maps any real number to its factorial | yes | no |

## Non-functional transformations.

| transformation | description | why isn't it a function? |
| -------------- | ----------- | ----------------------- |
| `rand.Int()` | maps the current state of a pseudo-random number generator to an integer | subsequent calls can have different outputs |
| `time.Now()` | maps the current state of the computer's clock to a time | subsequent calls can have different outputs |

- `rand.Int()` is not a function, since it returns a different value each time it's called.
- The mathematical operation `square root` is not a function, since (-2*-2) = 4, but (2*2) = 4 as well. The function `sqrt` that always chooses the positive root is a function, though.
- `time.Now()` is not a function, since it depends on the state of a piece of hardware (the computer's clock).
- `rand.Int()` can be mapped to a function by passing in the current state of a pseudo-random number generator.
- `square root` can be mapped to a function by always choosing the positive root.


  - "functional programming": 
  - function can be replaced by its value
  - functions with same I/O are equivalent
    - stateless transformations 
    - no room for side effects
- functions as objects
- what's the function signature of main?
