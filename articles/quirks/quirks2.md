# Golang Quirks & Tricks, Pt 2

#### A Programming Article by Efron Licht

#### Feb 2023

Go is generally considered a 'simple' language, but it has more edge cases and tricks than most might expect. In my [last article](./quirks.html), we covered intermediate topics, like declaration, control flow, and the type system. Now we're going to get into more advanced topics: concurrency, `unsafe`, and `reflect`.

By their nature, these articles are somewhat of a grab-bag without unifying theme, but the extremely positive response to the last one has convinced me they're worthwhile as a kind of whirlwind tour of more advanced topics.

As before, I'll link to the Go spec where appropriate. Most code examples link to a demonstration on the go playground.


##### more articles

- advanced go & gamedev

  1. [advanced go: reflection-based debug console](https://eblog.fly.dev/console.html)
  2. [reflection-based debug console: autocomplete](https://eblog.fly.dev/console-autocomplete.html)

- go quirks & tricks

  1. [declaration, control flow, typesystem](https://eblog.fly.dev/quirks.html)
  2. [concurrency, unsafe, reflect](https://eblog.fly.dev/quirks2.html)
  3. [arrays, validation, build constraints](https://eblog.fly.dev/quirks3.html)

- starting software

  1. [start fast: booting go programs quickly with `inittrace` and `nonblocking[T]`](https://eblog.fly.dev/startfast.html)
  1. [docker should be fast, not slow](https://eblog.fly.dev/fastdocker.html)
  1. [have you tried turning it on and off again?](https://eblog.fly.dev/onoff.html)
  1. [test fast: a practical guide to a livable test suite](https://eblog.fly.dev/testfast.html)

- miscellaneous
  1. [faststack: analyzing & optimizing gin's panic stack traces](https://eblog.fly.dev/faststack.html)
  1. [simple byte hacking: a uuid adventure](https://eblog.fly.dev/bytehacking.html)

### generics

You can write a generic that neither takes or returns a value of it's type parameter. We did this in the previous article to 'tag' a zero-sized type with an associated type:

```go
type contextKey[T any] struct{}
```

But it can be nice for making convenience wrappers around some of the functions in `reflect` and `unsafe`, too. I'll show an example, but first, a bit of background:

## type inference, & interfaces

Go will _infer types_ under some circumstances; most notably, go will happily convert a concrete type to an interface it satisfies: (like `int` -> `any` during `json.Marshal`), or an interface to a less restrictive interface (like `http.ResponseWriter` -> `io.Writer` during `fmt.Fprintf`) at a callsite or during an assignment.
This happens _before the function call_. This is why you can't ever get an `interface` type when you call `reflect.TypeOf`:

```go
func main() {  // https://go.dev/play/p/io4pUcl2oiS

 for _, t := range []any{
  "",
  new(string),
  any(nil),
  io.Writer(new(bytes.Buffer)),
  io.Writer(io.Discard),
  (*io.Writer)(&io.Discard),
  (*any)(nil),
 } {
  fmt.Println(reflect.TypeOf(t))
 }
}
```

```
string
*string
<nil>
*bytes.Buffer
io.discard
*io.Writer
*interface {}
```

But wait: right here it seems like we have two interfaces: `io.Writer` and `interface{}`! Not quite. We have _pointers_ to interfaces, which are _concrete_ types.
That's because a `(*io.Writer)` is converted to an `any` with concrete type `*io.Writer`, which is itself a pointer to a pointer to an io.Writer, which is _itself_ a pointer.

Let's walk through step-by step

```go
var buf bytes.Buffer
var bufp *bytes.Buffer = &buf // explicit:      1 ptr
var w io.Writer = &buf // implicit conversion to `io.Writer<&bytes.Buffer>`; 2 ptrs
var pw *io.Writer = &w // explicit:  3 ptrs
var t reflect.Type = reflect.TypeOf(pw) // implicit conversion to `any<*io.Writer>`: 4 ptrs
```

We can then use [`reflect.Type`](https://pkg.go.dev/reflect#Type).Elem() to get the actual type we're looking for. Let's wrap it up in the promised generic:

```go
// return the reflect.Type corresponding to the type parameter.
// this is an easier way to get at interface types and never allocates.
func typeOf[T any]() reflect.Type {
    return reflect.TypeOf((*T)(nil)).Elem()
}
func main() { // https://go.dev/play/p/qSqTgwVz6Wv
 fmt.Println(typeOf[io.Writer]())
}

```

Speaking of generics & type inference,

## generic type inference

Go can occasionally partially [infer the type of a generic function or struct](https://go.dev/ref/spec#Type_inference). It works like this: starting from the leftmost type parameter, Go attempts inference type-by-type. As soon as it finds an ambiguity, it stops trying to infer types, leaving the rest up to the programmer. This can sound a little vague, so let's use an example:

Go can't type infer the following function at all, so any invocation has to fully spell out the type parameters:

```go
func As1[FROM, TO any](f FROM) TO {
     return *(*TO)(unsafe.Pointer(&f)) // we'll conver unsafe later in this article.
}

func main() { // https://go.dev/play/p/no9apPCbMAX
    b := As1[uint64, [8]byte](4)
    fmt.Println(b)
}
```

> out: [4 0 0 0 0 0 0 0]

But by swapping the order of TO and FROM in the type parameters, we can partially type infer:

```go
func As2[TO, FROM any](f FROM) TO { // https://go.dev/play/p/50PexocApZy
 return *(*TO)(unsafe.Pointer(&f))
}
func main() {
    b := As2[[8]byte](4)
     fmt.Println(b)
}
```

This is usually good, but can sometimes cause difficulties when go's other type inference rules come into play. For example, the second program is _undefined behavior_ on 32-bit architectures; since it's technically `b := As2[[8]byte, int]]`.

It's worth noting that Go doesn't take the left-hand side of an expression into consideration for generic type interference. While we might expect this to compile, it doesn't:

```go
var b [8]byte = As1(uint64(4))
```

> compiler error: `./prog.go:10:20: cannot infer TO (prog.go:12:9)`

Careful ordering of your type parameters may make the difference between a pleasant API and an excruciating one.

## select & channels

These points are an extension of [dave cheney's article, 'channel axioms'](https://dave.cheney.net/2014/03/19/channel-axioms)

## nil channels block forever

A nil channel cannot send or recieve but blocks forever. This is usually a bug, but can be used to our advantage.

For example, we could combine three channels into one without favoring any channel for input: [^1]

[^1:] for the purpose of the following N examples, we're going to have exactly three channels, and we'll have their elements be ints.
a proper implementation should probably be generic, since this is easy to mess up and you'd rather do it once.
additionally, because go doesn't let you easily select from a variadic amount of channels at runtime, you generally have to write these functions for each arity (that is, number of different channels) you need. This is not too hard to do with code generation, since branches are identical. I hope to make that a subject of a later article.

```go

// combine three channels one. the output channel is closed when all three inputs are.
func splice[T any](a, b, c <-chan T) chan T { // https://go.dev/play/p/ndtODuvmO2e
 dst := make(chan T)
 go spliceInto(dst, a, b, c)
 return dst
}


// splice the elements of a, b, and c into dst. no ordering is guaranteed, but this does not favor any input channel.
func spliceInto[T any](dst chan<- T, a, b, c <-chan T) {
LOOP:
 for a != nil || b != nil || c != nil {
  select {
  case t, ok := <-a:
   if !ok {
    a = nil
    continue LOOP
   }
   dst <- t
  case t, ok := <-b:
   if !ok {
    b = nil
    continue LOOP
   }
   dst <- t
  case t, ok := <-c:
   if !ok {
    c = nil
    continue LOOP
   }
   dst <- t
  }
 }
 close(dst)
}

```

Or balance inputs, getting exactly one element each from a, b, and c before moving on to the next 'round' of elements:

```go

// feed one element each from a, b, and c each into the returned channel.
// the ordering of those elements within the rounds is not guaranteed.
func gatherRoundRobin(a, b, c <- chan int) (dst chan int) { // https://go.dev/play/p/0ftVD526ZQQ
 dst = make(chan int, 3)
 go func() {
  for {
   gatherRound(dst, a, b, c)
  }
 }()
 return dst
}

func gatherRound(dst chan <- int,a, b, c <- chan int) bool {
 // note scope: since a, b, and c are copied when we call this function,
 // nilling them out here doesn't affect the outer scope.
 for a != nil || b != nil || c != nil {
  var n int
  select {
  case n = <-a:
   a = nil
  case n = <-b:
   b = nil
  case n = <-c:
   c = nil
  }
  dst <- n
 }
 return true
}

func main() {
 a := make(chan int, 3)
 b := make(chan int, 3)
 c := make(chan int, 3)
 for i := 1; i <= 3; i++ {
  a <- i
  b <- i * 10
  c <- i * 100
 }
 ch := gatherRoundRobin(a, b, c)
 for i := 0; i < 9; i++ {
  fmt.Println(<-ch)
 }
}
```

## you can select on a send as well as a receive

This can be handy for dividing work equally among a number of potential workers.

> output: `100 010 001 020 200 002 003 030 300` (but other outputs may be possible.)

**FOOTGUN WARNING**:
Select contains a number of footguns.

- You can't check if a channel is closed without attempting to receive from it. This can quickly lead you to throw away data.
- Only the _leftmost channel_ in a select statement is actually selected. In other words, this implementation of `gatherRound`, while pleasingly compact, quickly deadlocks:

```go
func gatherRoundBad(dst chan int, a, b, c chan int) bool { // https://go.dev/play/p/7w0N2NYNstA
 for a != nil || b != nil || c != nil {
  var n int
  select {
  case dst <- <-a:
   a = nil
  case dst <- <-b:
   b = nil
  case dst <- <-c:
   c = nil
  }
  dst <- n
 }
 return true
}
```

> `fatal error: all goroutines are asleep - deadlock!`

Digging into [Go's spec](https://go.dev/ref/spec#RecvExpr), we find that

> For all the cases in the statement, the channel operands of receive operations and the channel and **right-hand-side expressions of send statements are evaluated exactly once, in source order, upon entering the "select" statement**. The result is a set of channels to receive from or send to, and the corresponding values to send. **Any side effects in that evaluation will occur irrespective of which (if any) communication operation is selected to proceed**. Expressions on the left-hand side of a RecvStmt with a short variable declaration or assignment are not yet evaluated.

Go makes it easy to spawn concurrent tasks, but managing them is difficult to get right. I've only scratched the surface here, and these examples are trivial and don't properly handle cancellation, etc. Since the advent of generics, a variety of structured concurrency libraries have been popping up. With luck, soon they'll be robust enough we don'th have to do this kind of thing by hand.

## channel conversions

While you can convert a bidirectional channel to a one-directional channel without issue:

```go
func main() {
 a := make(chan int) // https://go.dev/play/p/r3JTWjam0rX
 var _ <-chan int = a
 var _ chan<- int = a
}
```

The same is not true of data structures _containing_ bidirectional channels, though such a transformation should always be safe, since a channel regardless of direction is just a pointer to a \*runtime.hchan structure.

```go
func main() { // https://go.dev/play/p/WSd4XO6AaSg
 s := []chan int{}
 var _ []<-chan int = s

 var m = make(map[string]chan int)
 var _ map[string]<-chan int = m
}
```

> compiler errors:

> ./prog.go:7:23: cannot use sliceOfChannels (variable of type `[]chan int`) as `[]<-chan int` value in variable declaration`
>
> ./prog.go:8:23: cannot use sliceOfChannels (variable of type `[]chan int`) as `[]chan<- int` value in variable declaration`

This is one of the few cases where we know better than the compiler. We can get around these restrictions via the `unsafe` package.

## unsafe

The unsafe package lets you reinterpret memory as you see fit, subverting go's type system entirely.

Turn one type into another with the same size and alignment with the following transformation:

```go
// return a shallow copy of the bytes of T as a B
func copyAs[B, T any](t T) (b B) {
 return *(*B)(unsafe.Pointer(&t))
}

func asReceivers[T any](chans []chan T) []<-chan T {
 return copyAs[[]<-chan T](chans)
}

func asSenders[T any](chans []chan T) []chan<- T {
 return copyAs[[]chan<- T](chans)
}
```

This is _wildly_ unsafe for a number of reasons, and this isn't the article to go into them. Still, it can be handy when _we_ know two types are the same, but can't convince the compiler.

The following transformations ARE safe, and occasionally useful:
| FROM | TO | BIDIRECTIONAL |
| --- | ---- | --- |
| `uintN` | `[N/8]byte` | ✅ |
| `[M]uintN` |`[M*(N/8)]byte`| ✅ |
| `[]chan  T`| `[]<- chan   T`| ❌ |
| `[]chan  T` | `[]chan <-  T`|❌ |
| `map[K]  chan T`| `map[K]<- chan   T`| ❌|
| `map[K] chan  T` | `map[K] chan <-  T`|❌ |

Note that these transformations work on ARRAYS, not slices.

## unsafe slice transformations

Transforming slices isn't too bad, though: (If you're not familiar with the internals of slices, see [this article on the go blog](https://go.dev/blog/slices-intro) or this [follow-up by dave cheney](https://dave.cheney.net/2018/07/12/slices-from-the-ground-up) first.)

Because slice lengths are known at _runtime_, we'll have to do a little bit of math, then generate the slice ourselves via the underlying pointer:

```go

func main() { // https://go.dev/play/p/c2dyEAD9aD-
 // create a slice of uint16s...
 sixteen := []uint16{0x0123, 0x3456}
 fmt.Printf("%T (before): %#04x\n", sixteen, sixteen)
 eight := SliceAs[uint8](sixteen) // and reinterpret them as uint8s
 fmt.Printf("%T (before): %#02x\n", eight, eight)
 // since they share the same array, a change to one...
 eight[1] = 0xA

 // is reflected in the other. note that most, but not all modern architectures are little-endian.
 fmt.Printf("%T (after): %#04x\n", sixteen, sixteen)
}

func SliceAs[B, T any](t []T) (b []B) {
 n := size[T]() * len(t)

 sizeB := size[B]()
 if n%size[B]() != 0 {
  panic(fmt.Errorf("can't convert %T to %T: out of bounds", t, b))
 }
 newLen := n / sizeB
 return unsafe.Slice((*B)(unsafe.Pointer(unsafe.SliceData(t))), newLen)
}
func size[T any]() int {
 var t T
 return int(unsafe.Sizeof(t))
}
```

> output:
>
> ```
> []uint16 (before): [0x0123 0x3456]
> []uint8 (before): 0x23015634
> []uint16 (after): [0x0a23 0x3456]
> ```

In general, the unsafe package is best avoided in production code, but sometimes you actually _do_ know better than the compiler. I encourage my readers to play around with the `unsafe` package on their own time to gain an intuition about how Go actually lays things out in memory. Make sure to read the [package](https://pkg.go.dev/unsafe) and [spec](https://go.dev/ref/spec#Package_unsafe) documentation carefully.

I hope this was helpful! I think this is the end of this series; I'm planning to do some deeper dives next time.

Like this article? Need help making great software, or just want to save a couple hundred thousand dollars on your cloud bill? Hire me, or bring me in to consult. Professional enquiries at
[efron.dev@gmail.com](efron.dev@gmail.com) or [linkedin](https://www.linkedin.com/in/efronlicht)
