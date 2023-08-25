### IN PROGRESS. IF YOU SEE THIS, IT'S NOT DONE YET. READ ONLY IF YOU'RE CURIOUS

# advanced go: reflection-based debug console

This article assumes an intermediate level of knowledge of Go, including at least the basics of reflection. You can learn about reflection via the [official docs on golang.org](https://golang.org/pkg/reflect/) or by reading [the laws of reflectionon the Go Blog].(<https://blog.golang.org/laws-of-reflection>).

## 1. Motivation

For the last few months I've been working nearly full-time on my own 2D Game, 'Tactical Tapir'. Working with games is a new domain for me: systems are not as isolable as they are in web development or device drivers. In some ways, the questions game development asks are much more subjective: instead of asking "is this code correct?", you ask "does this feel nice?"

Here's a non-exhaustive list of tasks I'd like to be able to do 'live':

- spawn an enemy or obstacle
- change player traits (health, movement speed, etc.)
- manipulate the positioning, size, or look of UI elements
- track the state of a variable or group of related variables from frame-to-frame.

## 2. An (incredibly brief) introduction to game development

A game is a program that runs in a loop. Each update, or `frame`, does the following:

- take player input
- update the state of the game using that input and the previous state
- draw the relevant part of that state to the screen.'

While we might do parts within `Update()` in parallel, _between_ frames, the State is fixed. We're 'safe' to modify that state arbitarily, at the beginning of each frame, since we know that nothing else is touching it. And we're safe to read from it after `Update()`, since we know that nothing else is touching it.

That is, our game loop should look like this:

```go
for tick := 0; ; tick++ {
    inputs := input.ThisFrame()
    debugUpdate(game, inputs)
    if err := game.Update(inputs, tick); err != nil {
        log.Fatalf("shutdown: update(): %v")
    }
    <-ticker // but wait for the next frame to draw
    if err := game.Draw(screen); err != nil {
        log.Fatalf("shutdown: draw(): %v")
    }
}
```

### 3: Naive cheats

Now that we know where to put them, let's write some cheats. Let's test the idea with two simple cheats: one to give the player infinite ammo, and one to give the player infinite health.

We'll use keyboard inputs to trigger the cheats. We don't want the player to trigger them accidentally, so we'll require that they hold down the `shift` and `ctrl` keys while pressing `H` and `A` respectively.

```go
func applyCheats(g *Game, input Inputs) {
    if input.Held[KeyShift] && input.Held[KeyCtrl] {// check for cheats: if no ctrl+shift, no cheats
        if input.JustPressed[KeyA] {
            log.Println("infinite ammo")
            g.Player.Ammo = math.Inf(1)
        }
        if input.JustPressed[KeyH] {
            log.Println("infinite health")
            g.Player.HP = math.Inf(1)
        }
    }
}
func (g *Game) Update(input Inputs) {
    applyCheats(g, input)
    // ...rest of update
}
```

This works pretty well. In fact, we can visualize it as a kind of table:
|cheat|key|description|
|---|---|---|
|∞ ammo|`ctrl`+`shift`+`A`|set `State.Player.Ammo` to math.Inf(1)|
|∞ hp|`ctrl`+`shift`+`H`|set `State.Player.HP` to math.Inf(1)|

Which naturally suggests using a map to store the cheats:

```go
var cheats map[Key]struct { 
    description string
    apply func(*Game)
} {
    KeyA: {
        description: "spawn ammo",
        apply: func(g *Game) {
            g.Pickups = append(g.Pickups, AmmoPickup{...})
        },
    },
    KeyH: {
        description: "spawn health",
        apply: func(g *Game) {
            g.Pickups = append(g.Pickups, HealthPickup{...})
        },
    },
}
func applyCheats(g *Game, input Inputs, cheats map[Key]struct {
    description string
    apply func(*Game)
}) {
    if input.Held[KeyShift] && input.Held[KeyCtrl] {// check for cheats: if no ctrl+shift, no cheats
        for key, cheat := range cheats { 
            if input.JustPressed[key]
                log.Println(cheat.description)
                cheat.f(g)
            }
    }
}

func (g *Game) Update(input Inputs) {
    applyCheats(g, input, cheats)
    // ...rest of update
}
```

While this works well for some tasks, a few limitations are immediately apparent:

- Every new field we touch requires it's own function, even if we have multiple fields that are logically related.
- Those functions must be niladic: that is, they don't take any arguments, they just modify the gamestate directly.
- So for each field, and for each value we want to set it to, we need a separate function.

For some things, like 'infinite HP', this is fine. But for other things, like 'exactly 28 HP', this is absurd. You'd either need to write a function for every possible value, or expand the system somehow to take arguments. We need a more general approach.

### 4: A more general approach

The default API for general-purpose inputs is the command line. That is, prompt that takes a line of text, parses it as a command, and executes it.

A basic API might look like this:
|command|description|example| example output |
|---|---|---|---|
| `<field> = <value>` | set the field to the value | `player.hp = 100` | player.hp -> 100
| `print <field>`| print the value of the field | `print player.hp` | player.hp: 100

Let's see how we might implement this.

A traditional CLI is pretty simple: we grab input and put it into a buffer until we hit a newline. In the C tradition, it might look like this:

 For now, I'm going to ignore rendering the prompt, and just focus on parsing the input.

```go
type Prompt struct {
    Cursor int
    Text string // current line of text.
    buf []byte // underlying buffer; should have the contents of Text.
}
func (p *Prompt) Update(input Input) (line string, pressedEnter bool) {
    if p.Cursor < 0 {
        p.Cursor = 0
    } else if p.Cursor > len(p.Text) {
        p.Cursor = len(p.Text)
    }

    var 
    for _, p := range input.PrintableThisFrame() {
        buf = append(buf, p)
    }
    switch {
        case input.JustPressed[KeyEnter] && p.Text != "": // return the current line of text
            line := p.Text
            p.Cursor, p.Text = 0, "" // 
            return line, true
        case input.JustPressed[KeyBackspace] && p.Cursor > 0: // delete the character before the cursor
            p.Text = p.Text[:p.Cursor-1] + p.Text[p.Cursor:]
            p.Cursor--

        

    }
}
```

- Print a prompt to the screen that fills in with the user's input.
- Parse that input into a command when the user presses enter.
- Debug access should be opt-out, not opt-in.

The first game I played with a Debug Console was 1996's quake: TODO: PICTURE
And it's still a staple of modern games

- Factorio <TODO: PICTURE>
- Doom 2016  <TODO: PICTURE>

Many game developers inject a LUA interpreter for this reason. LUA is a fast & flexible language with an incredibly small runtime, and there may be pre-existing Go bindings for it - if you're in this situation, you may want to start there.

## 4: Reflection

## conceptual breakdown

We have four separate problems to solve.

- Parsing user input into executable commands, usually in the form `op <path> <value>`. See the previous section for more details.

The remaininng three problems all require reflection:
- Resolving paths into a specific field of a struct or slice.
- Converting the user-provided value into the correct type.
- Setting the field to the new value.

The first one is relatively straightforward, but the last three will require reflection. Reflection isn't commonly used in user code, so it's worth reviewing the basics.

### reflection / refresher

Reflect allows you to operate on Go values of arbitrary type without knowing what type or types they are ahead of time. I'll first show a few examples of what you can do with it, then present a cheatsheet of the most useful types and functions for you to refer to, then we'll get back to the console.

#### example: set or get the value of a field of any numeric type

```go
// https://go.dev/play/p/gh7TMf2-JlE

var f64type = reflect.TypeOf(0.0)   
//  get the value of "`X`" and "`Y`" fields of a struct, regardless of what type the struct is, as long as they're both _any_ numeric type, even if X or Y are embedded in another struct.
func getXY(v reflect.Value) (x, y float64, ok bool) {   
    if v.Type().Kind() != reflect.Struct { // make sure we have a struct
        return 0, 0, false
    }
    // check if v.X or v.Y would be valid expressions at compile time on the type of v
    vx, vy := v.FieldByName("X"), v.FieldByName("Y")

    if !vx.IsValid() || !vy.IsValid() { 
        // they're not, so we can't do it at runtime either
        return 0, 0, false
    }
    // and that f64(v.X) and f64(v.Y) would be valid conversions at compile time
    if !vx.CanConvert(f64type) || !vy.CanConvert(f64type) {
        // they're not, so we can't do it at runtime either
        return 0, 0, false

    }
    x, y = vx.Convert(f64type).Float(), vy.Convert(f64type).Float()
    return x, y, true
}
// set the value of the "`X`" and "`Y`" fields of a struct, so long as X and Y are both _any_ numeric type, even if X or Y are embedded in another struct.
// we could use this to, for example, set the position of an object in a game to the position of the mouse cursor.
func setXY(v reflect.Value, x, y float64) bool {
    if v.Type().Kind() != reflect.Struct {
        return false // not a struct
    }
    vx, vy := v.FieldByName("X"), v.FieldByName("Y")
    if !vx.IsValid() || !vy.IsValid() {
        return false // no X or Y field
    }
    if !vx.CanSet() || !vy.CanSet() {
        return false // X or Y is unexported, part of an unexported struct, or isn't in an addressable struct
    }
    if !f64type.ConvertibleTo(vx.Type()) || !f64type.ConvertibleTo(vy.Type()) {
       return false
    }
    vx.SetFloat(x)
    vy.SetFloat(y)
}
```

#### IN

```go
// https://go.dev/play/p/gh7TMf2-JlE
func main(){
    for _, v := range []any{
        &image.Point{1, 2}, // X and Y are `int` in this package!
        &struct{ X, Y float64 }{3, 4},
        &struct{ image.Point }{image.Point{5, 6}},
    } {
        v := reflect.ValueOf(v).Elem()                // get the Value of the pointer
        x, y, _ := getXY(v)                           // get the value of the X and Y fields as float64s
        fmt.Printf("%s: %v", v.Type(), v.Interface()) // print the type and the values
        setXY(v, x*10, y*10)                          // set the value of the X and Y fields to 10x their original value
        fmt.Printf("-> %v\n", v.Interface())          // print the type and the values
    }
}
```

OUT:

```
image.Point: (1,2)-> (10,20)
struct { X float64; Y float64 }: {3 4}-> {30 40}
struct { image.Point }: (5,6)-> (50,60)
```

#### EXAMPLE: zero out any field of any struct

```go
// zero out the given field of a struct, regardless of the type of struct or field, or whether the field is embedded in another struct.
func zeroField(v reflect.Value, fieldName string) bool {
   if v.Type().Kind() != reflect.Struct {
        return false // not a struct
    }
    f := v.FieldByName(fieldName)
    if !f.IsValid() {
        return false // no field
    }
    if !f.CanSet() {
        return false // field is unexported, part of an unexported struct, or isn't in an addressable struct
    }
    f.Set(reflect.Zero(f.Type()))
    return true
}

```

IN:

```go
// https://go.dev/play/p/YO8LmQqqZuJ
func main() {
 type A struct{ F string }
 var a = A{"foo"}
 fmt.Printf("a: before: %+v\n", a)
 zeroField(reflect.ValueOf(&a).Elem(), "F")
 fmt.Printf("a: after: %+v\n", a)

 type B struct{ F int }
 var b = B{2}
 fmt.Printf("b: before: %+v\n", b)
 zeroField(reflect.ValueOf(&b).Elem(), "F")
 fmt.Printf("b: after: %+v\n", b)

}
```

OUT:

```
a: before: {F:foo}
a: after: {F:}
b: before: {F:2}
b: after: {F:0}
```

#### types and values

Get a Value from a normal variable via `reflect.ValueOf(t)`, then modify it with the various functions on `reflect.Value`. Pretty much anything you can do in 'ordinary' go you can do with some combination of `reflect.Value`'s methods. E.g, the following snippets are functionally equivalent:

```go
var n int
reflect.ValueOf(&n).Elem().SetInt(50)
```

```go
func main() {var n int; *(&n) = 50}
```

Or to show it another way:

```go
reflect.ValueOf(&n). // &
Elem(). // *
SetInt(50) // =
```

**Note the pointers**. Since `reflect.ValueOf` is an ordinary function, you'll need to **pass a pointer** if you want to modify one of the arguments, just like any other function.

Find out information about a type via `reflect.TypeOf(t)` or the underlying primitive type via `Type.Kind()`.

In the following notation, eleme `t` is a [`reflect.Type`](https://pkg.go.dev/reflect#Type), `v` is a [`reflect.Value`](https://pkg.go.dev/reflect#Value), `T` and `B` is are types, and `t` and `b` are values of those types (not `reflect.Values`, but the normal type you get via `':='`, `'var'`, etc.

## cheatsheet

Here's a quick cheatsheet of the types and functions we'll use in this article. Feel free to skip this for now, and come back to it when or if you need it.

| shorthand | type | obtained via  |
|---|---|---|
|v | [`reflect.Value`](https://pkg.go.dev/reflect#Value) | `reflect.ValueOf("some string")` |
| t | [`reflect.Type`](https://pkg.go.dev/reflect#Type) | `v.Type()` or `reflect.TypeOf("another string")` |
| k | [`reflect.Kind`](https://pkg.go.dev/reflect#Kind) | `t.Kind()` |
| f | [`reflect.StructField`](https://pkg.go.dev/reflect#StructField) | `t.Field()` or `t.FieldByName()` or `t.FieldByNameFunc()`
| n | `int8..=int64` or `int` | `n := 2` |
| b | `bool` | `b := true` |
| s | `string` or `struct` | `s := "some string"`, `s := struct{foo int}{"foo}` |
| m | `map` | `m := map[string]int{"a": 1}` |
| a | `slice` or `array` | `a := []int{1, 2, 3}` |

| function | description | example | analogous to
|---|---|---|---|
|[`ValueOf`](https://pkg.go.dev/reflect#ValueOf)| get a [`Value`](https://pkg.go.dev/reflect#Value) from an ordinary value | `reflect.ValueOf(int(2))` | `t := 2` |
|[`TypeOf`](https://pkg.go.dev/reflect#TypeOf)| get a [`Type`](https://pkg.go.dev/reflect#Type) from the value | `t := reflect.TypeOf(int(2))` | `int`|
|[Type.Kind](https://pkg.go.dev/reflect#Type.Kind) | get the underlying primitive type | `t.Kind()` | `int` |
|---|---|---|---|
|[`Type.ConvertibleTo`](https://pkg.go.dev/reflect#Type.ConvertibleTo) | can the type be converted to a different type? | `t.ConvertibleTo(reflect.TypeOf(0))` |
|[`Value.Addr`](https://pkg.go.dev/reflect#Value.Addr) | get the address of a value | `v.Addr()` | `&t` |
|[`Value.CanAddr`](https://pkg.go.dev/reflect#Value.CanAddr) | can the value be addressed? | `v.CanAddr()` | |
|[`Value.CanConvert`](https://pkg.go.dev/reflect#Value.CanConvert) | can the value be converted to a different type? | `v.CanConvert(reflect.TypeOf(0))` ||
|[`Value.Convert`](https://pkg.go.dev/reflect#Value.Convert) | convert a value to a different type | `reflect.ValueOf(&t).Elem().Convert(reflect.TypeOf(b))` | `T(v)` | | use
|[`Value.Elem`](https://pkg.go.dev/reflect#Value.Elem) | dereference a pointer or interface | `v.Elem()` | `*t` | |
|[`Value.Field`](https://pkg.go.dev/reflect#Value.Field) | get the `nth` field of a struct | `v.Field(0)` |
|[`Value.FieldByName`](https://pkg.go.dev/reflect#Value.FieldByName) | for `struct` kinds, get the field with the given name | `v.FieldByName("someField")` | `t.someField`
|[`Value.FieldByNameFunc`](https://pkg.go.dev/reflect#Value.FieldByNameFunc) | for `struct` kinds, get the field with the given name, matching the given predicate | `v.FieldByNameFunc(func(s string) bool { return strings.EqualFold(s, "somefield") })` | `s.someField` or `s.somefield` or `s.Somefield`
|[`Value.Index`](https://pkg.go.dev/reflect#Value.Index) | for `array` and `slice` kinds, get the `nth` element | `v.Index(0)` | `a[0]`
|[`Value.Interface`](https://pkg.go.dev/reflect#Value.Interface)| get an ordinary value back from a `Value` (as `any`) | `reflect.ValueOf(2).Interface().(int)` | `any(int(2)).(int)` |
|[`Value.Len`](https://pkg.go.dev/reflect#Value.Len) | for `array`, `map`, and `slice` kinds, get the length | `v.Len()` | `len(a)`, `len(m)`
|[`Value.MapIndex`](https://pkg.go.dev/reflect#Value.MapIndex) | for `map` kinds, get the value associated with the given key | `v.MapIndex(reflect.ValueOf("someKey"))` | `m["someKey"]`
|[`Value.Set`](https://pkg.go.dev/reflect#Value.Set) | set lhs to rhs, if they're the same `Type` | `v.Set(reflect.ValueOf(2))` | `t = 2` | |  

OK, that covers what we'll need for now. Let's get back to the console.

### Resolving paths

We'd like to be able to access fields of structs, indices of slices, and values of maps using a single, uniform syntax. Taking a cue from `lua`, we'll use `.` as our access operator: ResolvePath(root, "player.0as") should work whether player is a struct, a slice, or a map.

### converting values to the correct type

We'd like all of the following commands to work, without worrying about the type of the fields or the values:
they should "just work":

- `set player.hp 100`
- `set player.hp 100.0`
- `set player.hp player.x`
- `set player.pos npcs.0.pos`
- `set player.pos someuint8`

That is, we have two situations: either the user-provided value is a literal, or it's a path. Let's start with literals.

#### converting literals

How to handle literals depends on the type of the field we're setting.

- **strings** require no processing.
- **numbers** can be treated as floats, and then converted to the correct type using `reflect.Value.Convert`. This loses some precision, but if it's good enough for javascript, it's good enough for us.
- **bools** can be parsed using `strconv.ParseBool`.
- **other types** can use the `encoding.TextUnmarshaler` interface, which is implemented by many types in the standard library, including `*time.Time` and `net.IP`. A note here: most of the time, these types require a _pointer_ for the method, so we might occasionally need to add a level of indirection to satisfy the interface.

Let's see what this looks like in code:

```go
// https://go.dev/play/p/xB6o7TAmuwn
var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
// part of a software article by Efron Licht.
package main

import (
 "encoding"
 "fmt"
 "net"
 "reflect"
 "strconv"
)

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

// SetString interprets src as a string literal, and attempts to set dst to that value.
// Conversions happen in this order:
// If dst, &dst, *dst, **dst, etc implement encoding.TextUnmarshaler, use UnmarshalText([]byte(src))
// Otherwise, if dst is a string, set it to src.
// Otherwise, if dst is a bool, set it to the result of strconv.ParseBool(src)
// Otherwise, if dst is a numeric type, set it to the result of strconv.ParseFloat(src, 64).
func SetString(dst reflect.Value, src string) error {
 // let's make sure we didn't dereference too far.
 if dst.CanAddr() { 
  dst = dst.Addr()
 }
 for i := 0; dst.Kind() == reflect.Ptr || dst.Kind() == reflect.Interface; i++ {
  if dst.Type().Implements(textUnmarshalerType) {
   return dst.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(src))
  }
  dst = dst.Elem()
  if i > 32 {
   panic("dereferenced 32 pointers, but still got a pointer or interface")
  }
 }
 switch dst.Kind() {
 default:
  return fmt.Errorf("cannot convert %s to %s", src, dst.Type())
 case reflect.String:
  dst.SetString(src)
  return nil
 case reflect.Bool:
  b, err := strconv.ParseBool(src)
  if err != nil {
   return fmt.Errorf("cannot convert %s to %s", src, dst.Type())
  }
  dst.SetBool(b)
  return nil
 case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
  reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
  reflect.Uintptr, reflect.Float32, reflect.Float64:
  f, err := strconv.ParseFloat(src, 64)
  if err != nil {
   return fmt.Errorf("cannot convert %s to %s: %w", src, dst.Type(), err)
  }
  dst.Set(reflect.ValueOf(f).Convert(dst.Type()))
  return nil
 }
}
```

Let's try it out on each of our cases:

```go
func main() {
 ip := new(net.IP) // implements encoding.TextUnmarshaler
 if err := SetString(reflect.ValueOf(ip), "192.168.1.1"); err != nil {
  panic(err)
 }
 fmt.Println(ip) // 192.168.1.1

 n := 0
 if err := SetString(reflect.ValueOf(&n), "22"); err != nil {
  panic(err)
 }
 fmt.Println(n) // 22

 s := "foo"
 if err := SetString(reflect.ValueOf(&s), "somestring"); err != nil {
  panic(err)
 }
 fmt.Println(s) // somestring

 b := false
 if err := SetString(reflect.ValueOf(&b), "true"); err != nil {
  panic(err)
 }
 fmt.Println(b) // true
}
```

Seems good. Let's handle paths.

#### converting paths

Paths are slightly more complicated. We need to resolve the path, and then convert the value at the end of the path to the correct type. Unlike literals, we don't want to 'stringly type', but we would like to allow for [go's usual type conversions](https://go.dev/ref/spec#Conversions), such as `int32` to `float64` or `[]byte` to `string`. [`reflect.Value.Convert`](https://pkg.go.dev/reflect#Value.Convert) and [`reflect.Value.CanConvert`](https://pkg.go.dev/reflect#Value.CanConvert) will do this for us.

We've already written `resolvePath`, so let's write a function to convert the value at the end of a path:

```go
// Set the value of dst to the value of src. If src is not convertible to dst, return an error.
func SetVal(dst, src reflect.Value) error {
    dst, src = deref(dst), deref(src)
    if src.ConvertibleTo(dst.Type()) {
        lhs.Set(rhs.Convert(dst.Type()))
        return nil
    }
    return fmt.Errorf("cannot convert %s to %s", src.Type(), dst.Type())
}
```

We can use this to implement `set`:

```go
func Set(root reflect.Value, dstPath, litOrSrcPath string) error {
    dst, err := ResolvePath(root, dstPath)
    if err != nil {
        return err
    }
    if strings.HasPrefix(litOrSrcPath, `"`) && strings.HasSuffix(litOrSrcPath, `"`) {
        // it's a literal. we need to parse it into the correct type.
        // we'll use the type of the lhs as a guide.
        return SetString(dst, litOrSrcPath[1:len(litOrSrcPath)-1])
    }
    src, pathErr := ResolvePath(root, litOrSrcPath)
    if pathErr != nil {
        // maybe it's a literal?
        if litErr := SetString(dst, litOrSrcPath); litErr != nil {
            return fmt.Errorf("set %s %s: %s not a path, and could not be parsed as a literal: %w", dstPath, litOrSrcPath, litErr)
        
    }

    return SetVal(dst, src)
}
```
fResolve
Let's start with two basic commands: `set` and `print`. `set` will set a field to a value, and `print` will print the value of a field.

| op [args] | description | example|
|---|---|---|
| `set` | `set <path> <literal_or_path>` | `set player.health 100` |
| `print` | `print <path>` | `print player.health` |

Let's assume we have a complete line of user input:

```go
// note the generic here: we want to be able to set any field on any struct,
// but to do so we need an addressable instance. by making sure the root is a pointer,
// non-nil root must be an addressable instance.
func exec[T any](pt *T, line string) (logMsg string, err error) {
    defer func() { 
        // the debug console should never crash the game.
        if p := recover(); p != nil {
            err = fmt.Errorf("panic: %v", p)
        }
    }
    root := reflect.ValueOf(pt).Elem() // get the reflect.Value of the dereference of pt
    // split the line on whitespace
    args := strings.Fields(line)
    op, args := args[0], args[1:]
    switch op {
    default:
        return "", fmt.Errorf("unknown op %s", op)
    case "print":
        if len(args) != 1 {
            return "", fmt.Errorf("print: expected 1 argument, got %d", len(args))
        }
        v, err := resolvePath(root, args[0])
        if err != nil {
            return "", err
        }
        return fmt.Sprintf("%s = %v", args[0], v), nil
    case "set":
        if len(args) != 2 {
            return "", fmt.Errorf("set: expected 2 arguments, got %d", len(args))
        }
        lhs, err := ResolvePath(root, args[0])
        if err != nil {
            return "", err
        }
        // two cases: either the rhs is a literal, or it's a path.
        // if it's surrounded by quotes, it's DEFINITELY a literal.
        // if it's not, it MAY be a literal, but it may also be a path.
        if strings.HasPrefix(args[1], `"`) && strings.HasSuffix(args[1], `"`) {
            // it's a literal. we need to parse it into the correct type.
            // we'll use the type of the lhs as a guide.

        }
        rhs, err := ResolvePath(root, args[1])
        if 

     
    }
}
```

- We want to be able to set variables without worrying too much about types. That is, we want to be "stringly typed":

-

```
set player.health 100
```

should succeed regardless of whether player.health is an int16 or float32.

- By default, we should be able to set any field on any struct, but we need some way to filter out fields that are too 'fragile' to be modified at runtime. Go already has a convention for this: fields that start with a lowercase letter are not exported, and cannot be accessed outside of the package. It turns out the reflect package respects this convention, and will not allow you to set an unexported field. Since the GameState struct is at the "root" of our program, in main, and nothing else will actually import it, we can just use capital letters to denote fields that are safe to modify at runtime.

```go
type Gamestate struct {
    Player struct {
        Health int
        Ammo int
        X, Y int
        dangerousField unsafe.Pointer // this field is not exported, and cannot be modified at runtime
    }
}
```

### bonus: combining reflect and unsafe for true arbitrary modification



```go
func SetUnsafeValue(dst, src reflect.Value) {
    if src.Type().Size() != dst.Type().Size() {
        panic(fmt.Sprintf("cannot set %v (size %d) to %v (size %d)", src, src.Type().Size(), dst.Type(), dst.Type().Size()))
    }
    copy(
        unsafe.Slice((*byte)(dst.Addr().UnsafePointer()), int(dst.Type().Size())),
        unsafe.Slice((*byte)(src.Addr().UnsafePointer()), int(src.Type().Size())),
    )
}
// SetUnsafe sets the value of dst to the value of src, without obeying the usual rules about
// type conversions, field & type visibility, etc. Go wild.
// dst must be an addressable Value with a type that is the same size as src.
func SetUnsafe[T any](dst reflect.Value, src *T) {
	size := unsafe.Sizeof(*src)
	if size != dst.Type().Size() {
		panic(fmt.Sprintf("cannot set %v (size %d) to %v (size %d)", src, size, dst.Type(), dst.Type().Size()))
	}
	copy(
		unsafe.Slice((*byte)(dst.Addr().UnsafePointer()), int(size)),
		unsafe.Slice((*byte)(unsafe.Pointer(src)), int(size)),
	)
}

// SetUnsafeBytes sets the value of dst to the value of src, without obeying the usual rules about type conversions, field & type visibility, etc. 
// dst must be an addressable Value with a type that is the same size as the length of src (but it DOESN'T have to be conventionally settable).
//len(src) must be equal to the size of dst, or it will panic.
func SetUnsafeBytes(dst reflect.Value, src []byte) {
	if uintptr(len(src)) != dst.Type().Size() {
		panic(fmt.Sprintf("cannot set %v (size %d) via slice of len %d", dst.Type(), dst.Type().Size(), len(src)))
	}
	copy(
		unsafe.Slice((*byte)(dst.Addr().UnsafePointer()), len(src)),
		src,
	)

}