# Golang Quirks & Intermediate Tricks, Pt 1: Declarations, Control Flow, & Typesystem

#### A programming article by Efron Licht

#### Written Feb 2023
## [more articles](https://eblog.fly.dev)

- [bytehacking](./bytehacking.html)
- [tale of two stacks](./faststack.html)
- [go quirks & tricks](./quirks.md)
#### On: Go, Programming Languages

Go is generally considered a 'simple' language, but it has more edge cases and tricks than most might expect.

You can be a productive go programmer without knowing about or using most or any of these tricks, but some of them are pretty handy. This is part 1 of what I hope to be a continuing series.

## multi-statement lines with semicolons

Go is secretly a C-like language that terminates statements with semicolons. [The semicolons are actually _inserted_ early in compilation (during lexing)](https://go.dev/doc/effective_go#semicolons). This means you can put multiple statements on the same line by inserting semicolons!

Be warned: `gofmt` will usually break them up into multiple lines. In fact, you can _never_ have a single-line conditional. (Sorry, ternary conditional fans.)

Still, it can be handy for _really_ small two-statement functions, like in tests:

```go
func asJson(v any) []byte {b, _ := json.Marshal(v); return b}
```

## methods as functions ("method expressions")

Go methods are just functions. Given a type and method:

```go
type Point struct{X, Y float64}
func (p Point) Add(q Point) Point{}
```

You can call the method in the 'usual' way by providing a receiver and using `receiver.funcName(arg1, arg2, ...)`

```go
p, q := Point{1, 1}, Point(2,2)
r := fmt.Println(p.Add(q))
```

Or you can use the method as an ordinary, "bare" function via `typeName.funcName(arg0, arg1, arg2`)

```go
p, q := Point{1, 1}, Point(2,2); fmt.Println(p.Add(q))
```

This is called a [method expression](https://go.dev/ref/spec#Method_expressions). Unlike method calls, a method _expression_ won't automatically reference or de-reference a receiver for you, since there _is_ no receiver.

That is, while **this** code compiles fine,

```go
import "math/big"
var x big.Float
x.SetFloat64(10)
```

**This** code gives a compiler error:

```go
import "math/big"
var x big.Float
big.Float.SetFloat64(x, 10) // wrong
```

> invalid method expression big.Float.SetFloat64 (needs pointer receiver (*big.Float).SetFloat64)

The proper method expression is as follows:

```go
import "math/big"
var x big.Float
big.Float.SetFloat64(&x, 10) // right
```

Method expressions don't come up often, but they can occasionally save some work when you're sorting or deduping.

## `select` statements have `break`

`select` has no `continue`, but it _does_ have `break`. This can lead to nasty bugs if you're trying to break out of, say, an enclosing `switch` or loop. Use labels instead, as demonstrated by this code in the un-exported `filelock` package in go's stdlib:[^1]

[^1]: I'd love to talk more about the useful but un-exported tools in the go stdlib, like `strconv.appendQuotedWith`

```go
 // Wait until process Q has either failed or locked file B.
 // Otherwise, P.2 might not block on file B as intended.
locked:
 for {
  if _, err := os.Stat(filepath.Join(dir, "locked")); !os.IsNotExist(err) {
   break locked
  }
  select {
  case <-qDone:
   break locked
  case <-time.After(1 * time.Millisecond):
  }
 }
```

## go can infer the type of composite literals in some contexts, but not others

The following code gives a terse and unhelpful compiler error:

```go
type Q {A, B [3]int}
structOfArrays := Q{{}, {}}
fmt.Println(structOfArrays)
```

> missing type in composite literal

This implies that you always need to provide the types of composite literals, but that's just not true:

Go is happy to compile this without me spelling out the type of each item on the right-hand side:

```go
type S { N, M int}
arrayOfStructs := [3]S{{}, {}, {0, 1}})
fmt.Println(arrayOfStructs)
```

> [{0 0} {0 0} {0 1}]

Or even this:

```go
sliceOfMapOfArrayOfStructs := []map[string][2]S{{"foo": {{1, 2}, {}}}})
fmt.Println(sliceOfMapOfArrayOfStructs)
```

> [map[foo:[{1 2} {0 0}]]]

The actual rule is this: go will infer the types of composite literals if they're contained within an **array**, **map**, or **slice**, but struct fields and function arguments always need to spelled out explicitly.

There's a long-open [issue (#12584)](https://github.com/golang/go/issues/12854) hoping to address this inconsitency. I'd love to see more permissive composite literals.

## GOTO exists

The oft-maligned GOTO is an excellent piece of kit.
unlike many languages, Go's GOTO can only jump _forwards_, so it's hard to get yourself into the kind of trouble you could in 1980s BASIC. Use GOTO to avoid extraneous variables and conditionals and to jump out of blocks without having to define a function.

Speaking of which...

## you can make a block at any time

You don't need an `if`, `for`, `func`, or any other keyword to make a block.

```go
{
    name := "efron"
    {
        fmt.Println("hi ", name)
    }
}

```

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

By using a block here, we make it immediately clear that `less` is only going to exist for the context of this sort

We _could_ make a function for this, but that means jumping around, for us, the compiler, and the runtime (assuming it's not inlined).

Try starting with blocks, and promote them to functions when you find yourself needing to re-use the code.

But sometimes you do need a function, even for a single use:

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

This is the catchily-named "immediately-evaluated-function-expression", or IIFE for short. These are invaluable in languages which privelege functions over other kinds of blocks: for example, Javascript before it got the `let` keyword had no block scope, so you had to define functions every time you wanted a new namespace.

Go privleges functions over blocks in two ways:

- you can evaluate a function during variable declarations in the global namespace (that is, before `main()` or even `init()`, as shown in the example above.
- a `defer`-ed function evaluates at the end of the enclosing _function_ scope.

There's basically only two uses for IIFE's instead of blocks:

- for complex variable initialization in the global scope in a more natural way than using `init()`.
- if you need a function scope to use `defer()` or `recover()`, since

If you're going to use an IIFE with a return value, use the `var` declaration instead of `:=` - it makes it easier for the reader to understand the flow. And don't overdo it - you can always just define a closure and call it on the next line.


## you can declare types inside blocks

You can declare types inside any kind of block, but you can't declare methods on those types.
You _can_ define a function that takes that type using a function expression ("closure").

```go
func main() {
 type Point struct{ X, Y float64 }
 addPoint := func(p, q Point) Point {return Point{X: p.X + q.X, Y: p.Y + q.Y}}
 q := addPoint(Point{2, 3}, Point{-1, -1}))
 }
```

This obeys the ordinary block-scope rules:

```go
func main() {
    {type Point struct{X, Y float64}}
    var p Point 
}

    // this would be a compiler error: undefined: Point
    // fmt.Println(Point{1, 1})
```

This can make your code more straightforward. Just like variables, it's best to define a type as close to it's use and with as small of a scope as possible.

## go has anonymous structs

Sometimes you don't have to declare the type at all: go allows anonymous struct values. This is especially handy for functions like `json.Marshal` and `json.Unmarshal` which just depend on the _shape_ of the type.

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

Both anonymous structs and interfaces can be generic, too.

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

### Putting it together: A generic zero-sized type

As weird as it sounds, I have a use for a **zero-sized**, **generic** struct with unreachable members:\
`context.WithValue`.

Let's review the documentation:

> #### `func WithValue(parent Context, key, val any) Context`
>
> WithValue returns a copy of parent in which the value associated with key is
> val.
> Use context Values only for **request-scoped data that transits processes and
> APIs**, not for passing optional parameters to functions.
> The provided key must be comparable and should not be of type
> string or any other built-in type to avoid collisions between
> packages using context. **Users of WithValue should define their own
> types for keys. To avoid allocating when assigning to an
> `interface{}`, context keys often have concrete type
> `struct{}`.** Alternatively, exported context key variables' static
> type should be a pointer or interface.

Most **request-scoped data** is a singleton per request. That is, it doesn't make sense for a request to carry around multiple loggers, users, traces; you want to carry the _same one_ with you from function call to function call

The usual way Go programs have handled this is by making a separate context key per type you want to carry in the struct. But with the advent of generics in `go1.18`, instead of having to make a new zero-sized type for every struct, we can just make a single generic:

```go

// FromCtx returns the value of type T stored in the context, if any:
func FromCtx[T](ctx context) (T, bool) { t, ok := context.Value([0]T{}).(T); return t, ok)}
// WithValue returns a copy of parent in which the value associated with `CtxKey[T]{}` is
// val.
func WithValue[T](ctx context, t T)(context.Context) {return context.WithValue(ctx, [0]T{}, t)}
```

For fun, let's rewrite `FromCtx` as a truly hellish one-liner using (nearly) every trick we've learned so far:

```go
func FromCtx[T any](ctx context.Context) (T, bool) {t, ok := context.Context.Value(ctx, [0] struct{ _ func(T) };).(T);return t, ok}
```

That's right: this ugly SOB has a

- zero-sized type
- containing an anonymous `struct`
- with an unreachable member containing a function so it's not comparable
- declared inside a function
- used in a semi-colon terminated multi-statement line

### Next time(?)

- More contradictions and corner cases
- Advanced generics
- Unsafe
- Runtime shenanigans
