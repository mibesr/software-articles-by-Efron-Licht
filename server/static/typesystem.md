# Weird quirks in Go

- [Weird quirks in Go](#weird-quirks-in-go)
  - [go separates statements with semicolons](#go-separates-statements-with-semicolons)
  - [you can call a method without a receiver](#you-can-call-a-method-without-a-receiver)
  - [using bound methods as variables](#using-bound-methods-as-variables)
  - [go infers the type of r-values, but only in arrays and maps](#go-infers-the-type-of-r-values-but-only-in-arrays-and-maps)
  - [`select` statements have `break`](#select-statements-have-break)
  - [GOTO exists](#goto-exists)
  - [you can make a block at any time](#you-can-make-a-block-at-any-time)
  - [immediately-evaluated-function-expressions](#immediately-evaluated-function-expressions)
  - [you can declare types inside blocks](#you-can-declare-types-inside-blocks)
  - [go has anonymous structs](#go-has-anonymous-structs)
  - [... and anonymous interfaces](#-and-anonymous-interfaces)
    - [type aliases exist](#type-aliases-exist)
    - [generic type inference is weirdly order-sensitive](#generic-type-inference-is-weirdly-order-sensitive)
  - [zero-sized type](#zero-sized-type)
  - [unreachable struct members](#unreachable-struct-members)
  - [`const`-able types have restricted assignment rules](#const-able-types-have-restricted-assignment-rules)

## go separates statements with semicolons

Go is secretly a C-like language that terminates statements with semicolons. [The semicolons are actually _inserted_ early in compilation (during lexing)](https://go.dev/doc/effective_go#semicolons). This means you can put multiple statements on the same line by inserting semicolons! Be warned: most of the time `gofmt` will break them up into multiple lines anyways.

Still, it can be handy for _really_ small two-statement functions, like in tests:

```go
func asJson(v any) []byte {b, _ := json.Marshal(v); return b}
```

## you can call a method without a receiver

Go methods are just functions.
If I have a type `Point` with a method `Add`:

```go
type Point struct{X, Y float64}
func (p Point) Add(q Point) Point{}
```

I can call it in the usual way:

```go
fmt.Println(Point{1, 1}.Add(Point{2,2}))
```

> {3, 3}

OR I can use it as a bare function, called a "method expression":

```go
fmt.Println(Point.Add(Point{1, 1}, Point{2, 2}))
> {3, 3}
```

Unlike method calls, a method _expression_ the compiler won't automatically reference or de-reference a receiver for me, since there _is_ no receiver.

That is, while I can do this:

```go
import "math/big"
var x big.Float
x.SetFloat64(10)
```

This will give a compiler error:

```go
import "math/big"
var x big.Float
big.Float.SetFloat64(x) // wrong
```

> invalid method expression big.Float.SetFloat64 (needs pointer receiver (*big.Float).SetFloat64)


```go
import "math/big"
var x big.Float
big.Float.SetFloat64(&x) // right
```
Method expressions don't come up often, but they can occasionally save some work when you're sorting or deduping.
## using bound methods as variables

// TODO: this. example idea: point type, origin at non-zero, sortBy(points, p.AbsDist)

## go infers the type of r-values, but only in arrays and maps

// TODO: link to the appropriate issue

## `select` statements have `break`

`select` has no `continue`, but it _does_ have `break`. This can lead to nasty bugs if you're trying to break out of, say, an enclosing `switch` or loop. Use labels instead.

TODO: example from `estd`

## GOTO exists

The oft-maligned GOTO is an excellent piece of kit.
unlike many languages, Go's GOTO can only jump _forwards_, so it's hard to get yourself into the kind of trouble you could in 1980s BASIC. Use GOTO to avoid extraneous variables and conditionals and to jump out of blocks without having to define a function.

Speaking of which...

## you can make a block at any time

You don't need an `if`, `for`, `func`, or any other keyword to make a block. Blocks hint to the reader (and compiler) that variables will quickly- [Weird quirks in Go](#weird-quirks-in-go)
-restricted-assignment-rules)

I find this useful for complicated variable initialization:

```go
// fmtbench.go
// context: the variable sortBy is a command-line flag specifying the sort order. we've already validated it.

 { // sort results
  var less func(i, j int) bool
  switch *sortBy {
  default:
    goto PRINT
  case "allocs":
   less = func(i, j int) bool { return results[i].allocs < results[j].allocs }
  case "name":
   less = func(i, j int) bool { return results[i].name < results[j].name }
  case "runtime":
   less = func(i, j int) bool { return results[i].ns < results[j].ns }
  }
  sort.Slice(results, less)
 }
PRINT:
 for _, res := range results {
  fmt.Printf("|%s|%.3g|%.3g|%0.3g|%.3g|%0.3g|%.3g|%0.3g|\n", res.name, res.runs, res.ns, (res.ns/maxNS)*100, res.bytes, (res.bytes/maxBytes)*100, res.allocs, (res.allocs/maxAllocs)*100)
 }
```

By using a block here, we make it immediately clear that `less` is only going to exist for the context of this sort. Some might prefer to avoid the block and GOTO by using a function:

We _could_ make a function for this, but that means jumping around in the source. Not everything needs a function.

But if you _do_ need a function...

## immediately-evaluated-function-expressions
You can define a function and invoke it on the same line:


```go
import "crypto/rand"
import "encoding/binary"
var seed uint64 = func() uint64 {
    var b = make([8]byte)
    _, _  = rand.Read(b)
    return binary.LittleEndian.Uint64(b)
}()
```

This is the catchily-named "immediately-evaluated-function-expression", or IIFE for short. These are invaluable in languages which privelege functions over other kinds of blocks: for example, Javascript before it got the `let` keywordd had no block scope, so you had to define functions every time you wanted a new namespace.

Go privleges functions over blocks in two ways:

- you can evaluate a function during variable declarations in the global namespace (that is, before `main()` or even `init()`, as shown in the example above.
- a `defer`-ed function evaluates at the end of the enclosing _function_ scope.

There's basically only two uses for IIFE's instead of blocks:

- for complex variable initialization in the global scope in a more natural way than using `init()`.
- if you need a function scope to use `defer()` or `recover()`, since

If you're going to use an IIFE with a return value, use the `var` declaration instead of `:=` - it makes it easier for the reader to understand the flow. And don't overdo it - you can always just define a closure and call it on the next line.

## you can declare types inside blocks

You can declare types inside any kind of block, but you can't declare methods on those types.
You can make a closure that takes that type, though:

```go
func main() {
    {
 type Point struct{ X, Y float64 }
 addPoint := func(p, q Point) Point {
  return Point{X: p.X + q.X, Y: p.Y + q.Y}
 }
 fmt.Println(addPoint(Point{2, 3}, Point{-1, -1}))
    }
    // this would be a compiler error: undefined: Point
    // fmt.Println(Point{1, 1})
}
```

This is useful _all the time_. If you're only going to use a type in one function or block, declare it as close to it's use as possible.

## go has anonymous structs

But sometimes you don't have to declare the type at all: go allows anonymous struct values. This is especially handy for functions like `json.Marshal` and `json.Unmarshal` which just depend on the _shape_ of the type.

These anonymous structs can _nest_:

```go

var s struct {Name struct { First, Last string}}
json.Unmarshal([]byte(`{"Name": {"First": "efron", "Last": "licht"}, "Hobbies": ["pickleball", "techno", "kvetching"]}`,&s)
```

You can even make custom struct tags for your individual use case:

```go
// GET /health
import "json"
import "net/http"
func WriteHealth(w http.ResponseWriter, _ *http.Request) {
    json.NewEncoder(w).Encode(struct {
        Uptime time.Duration `json:"uptime"`
        Stats struct {
            Hits int64 `json:"hits"`
            Misses int64 `json:"misses"`
            Errors int64 `json:"errors"`
             Misses, Errors int64
        } `json:"stats"`
    })
}
```

## ... and anonymous interfaces

You _never_ have to declare the type of an interface: anywhere you can use `io.Writer`, you can use `interface{Write([]byte)(int, error)}`.

This can be handy for runtime specialization (that is, when you want to check if a type fulfills extra interfaces)

```go
import "gzip"
func writeZipped(w io.Writer, b []byte) (int, error) {
    zipw := gzip.NewWriter(w)
    n, err := zipw.Write(w)
    if err != nil {
        return n, err
    }
    if err := zipw.Close(); err != nil {
        return err
    }
    // flush the underlying buffer, if there is one
    if f, ok := w.(interface{Flush() error}); ok {
        _ = f.Flush()
    }
    // sync to disk, if it exists
    if f, ok := w.(interface{Sync() error}); ok {
        if f, ok := w.
    }
}
```

This is especially useful for function signatures. Suppose I'm going to call out to a database as part of a function.

(An aside: I don't particularly like mocking: I'd love to write an article about strategies you can use to avoid it).

The 'ordinary' function signature would look something like this:

```go
func SelectUser(ctx context.Context, db *sql.DB, userID uuid.UUID) (username string, createdAt time.Time, err error) {
    const query = `SELECT username, created_at FROM users where user.id = $1;`
    db.QueryRowContext(ctx, query, userID).Scan(&username, createdAt)
}
```

Suppose at some point we need to mock this for a test. *sql.DB is a struct, and it's not immediately apparent what we'd call the interface we'd replace it with. `DBer?` `QueryRowContexter`? In this case,  we can be clearest by omitting the name entirely: all the reader needs to know is that the DB has a function that looks like QueryRowContext().

We can make this mockable by just changing the function signature to use an _anonymous interface_.

```go
func SelectUser(
    ctx context.Context, 
    db interface{QueryRowContext(context.Context, string, ...args) *sql.Row},
    userID uuid.UUID
) (username string, createdAt time.Time, err error) {
```

I think the anonymous interface is actually _clearer_ than the named one for most single-method interfaces.

Both anonymous structs and interfaces can be generic, too, but I haven't figured out a use for that.

### type aliases exist

// TODO: This.

### generic type inference is weirdly order-sensitive

// TODO: some preamble; we're jumping straight into generics.
this compiles

```go
// copy the bytes of a T and reinterpret them as a B. wildly unsafe.
func copyAs[B, T any](t T) B { return *(*B)(unsafe.Pointer(&t)) }
 var _ = copyAs[[8]byte](uint64(0xFF))
```

But if we reverse the order of the declarations, it doesn't:

```go
// note: T before B
func copyAs[T, B any](t T) B { return *(*B)(unsafe.Pointer(&t)) }
var _ = copyAs[[8]byte](uint64(0xFF))
```

> cannot infer B

We have to spell out both halves of the generic:

```go
var _ = copyAs[uint64, [8]byte](uint64(0xFF))
```

./prog.go:13:25: cannot infer B (prog.go:10:16)

Somewhere down the line, we need to write a test that involves _mocking_ out this call. We can painlessly change the function signature using an _anonymous interface_:

```go
func SelectUser(
    ctx context.Context, 
    db interface{QueryRowContext(context.Context, string, ...any) *sql.Row},
    userID uuid.UUID
) (username string, createdAt time.Time, err error) {}
```

## zero-sized type

Empty structs or arrays with length zero take up no memory. These are most often used as handles to an interface, like `io.Discard`:

```go

## zero-sized types
// io/io.go

// Discard is a Writer on which all Write calls succeed
// without doing anything.
var Discard Writer = discard{}
type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil}
func (discard) WriteString(s string) (int, error) { return len(s), nil}
```

You can also use them as the value type in a map to make a set without allocating extra memory for values:

```go
var set = make(map[string][0]int)
```

but _don't do this_: `map[string]bool` is just as fast and has a much cleaner api.

You can get kind of silly with this:

```go
type cursed = [0]map[struct{}]struct{} // still zero-sized! don't do this.
```

## unreachable struct members

Structs can have unreachable member variables, specified using `_`. This has a few uses.

- You can use them to pad a struct to a specific size or alignment.

    This is occasionally handy for cool `unsafe` stuff like serializing or deserializing stuff straight from a bytestream without worrying about pesky stuff like 'safety'.

    ```go
    type Point struct{ X, Y, Z uint16 }
    type PaddedPoint struct {
    X, Y, Z uint16
    _       uint16
    }
    const format = "%12v\t%v\t%v\n"
    fmt.Printf(format, "type", "size", "align")
    fmt.Printf(format, "Point", unsafe.Sizeof(Point{}), unsafe.Alignof(Point{}))
    fmt.Printf(format, "PaddedPoint", unsafe.Sizeof(PaddedPoint{}), unsafe.Alignof(PaddedPoint{}))
    ```

    ```
            type size align
            Point    6     2
        PaddedPoint  8    2
    ```

- An unreachable member variable means you have to use the key-value notation to make a literal of that struct, _even within it's own package_.

    This means that if you add a field to the struct later, it's not a breaking change for users.  So as not to waste space, use a [`zero-sized type`](#zero-sized-type) for this.

    ```go
    type LogOptions struct {
        Level int8
        LogTime, LogFile, LogLine bool
        _ [0]int
    }
    ```

    Be careful with this: sometimes you _want_ changes to the API to be breaking changes, and changing the size of commonly-used types can have unforseen performance ramifications. It's best used for quality-of-life structs like the logging configuration above.
- Adding a field of uncomparable type makes the entire struct uncomparable.
    Structs comprised only of [comparable](https://go.dev/ref/spec#Comparison_operators) types (that is, ones where you can use the `==` operator) are themselves comparable. You may not want this to happen. Additionaly, the compiler has to generate a comparison function for each struct of this kind, which bloats the binary and increases compilation times. You can avoid this by having an unreachable member variable of an uncomparable type: the usual candidate is `[0]func()`, since it's zero-sized.

    ```go
        type NotComparable struct {
            N, M int
            _ [0]func()
        }
        var a, b NotComparable
        a == b
    ```

    >  invalid operation: a == b (struct containing [0]func() cannot be compared)
- Unreachable member variables can provide hints to tooling about how a type should be used. The most famous example of this is `copylock`. See [go issue #8005](https://github.com/golang/go/issues/8005#issuecomment-190753527) for more details.

## `const`-able types have restricted assignment rules

// TODO: fix this: reading the spec, I had a misconception here. Talk about named vs unnamed types.
Intermediate go progammers are probably familiar with the idea of 'untyped constants', like the ones in the `math` package.

If you're not, now might be a great time to review [Dave Cheney's article on Go consts](https://go.dev/blog/constants) on the official go blog.

It's valid for me to assign an `untyped float constant` to any `float` type, even though these assignments would be invalid for variables:

```go
const pi = 3.14159265358979323846264338327950288419716939937510582097494459
// this compiles
var f64 float64 = 2*pi
// and this compiles
var f32 float32 = 2*pi
// and THIS compiles
type MyFloat float64
var f MyFloat = pi
```

Even though none of these would compile for a variable:

```go
var pi float64 = 3.14159265358979323846264338327950288419716939937510582097494459
type MyFloat float64
var f MyFloat = pi
```

> cannot use pi (variable of type float64) as type MyFloat in variable declaration

How many kinds of `untyped const` value are there?

I would guess 5, corresponding to all the primitives we can declare a `const` for:

```go
const (
  b = true     // untyped bool
  s = "string" // untyped string
  c = 0 + 2i   // untyped complex
  f = 0.12     // untyped float
  n = 1        // untyped int
)
```

But this only works for `const`: if I declare a `var`, I can no longer assign across types:

```go
// in other words, this doesn't compile:
var n uint32 = 0
type my32 uint32
var _ my32 = n
```

This leads to an interesting dichotomy:
**variables** of "const-able" types (bool, string, complex, float32/64, int8..=64, uint8..=64) cannot be assigned to another label with a different type but the same underlying structural type.

```go
// in other words, this doesn't compile:
var n uint32 = 0
type my32 uint32
var _ my32 = n
```

Variables of _every other type_ can, but only if they're the 'base' type of that structure:

```go
// in other words, THIS compiles
 var b [8]struct{}
 type A [8]struct{}
 var _ A = b
// but this doesn't
    type B [8] struct{}
 type A [8]struct{}
    var b B
 var _ A = b
```

This isn't just true for arrays:  it's true for channels, maps, slices and pointers. For `struct`, you need to use an _anonymous struct_ of the same shape:

```go
// , THIS compiles
var s struct{N int}
type S struct{N int}
var _ S = s
// but this doesn't
type S struct{N int}
type Q struct{N int}
var s = S{2}
var _ Q = s
```

Is this _ever_ useful? I have no idea. weird, right?
