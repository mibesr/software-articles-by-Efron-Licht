

# reflect cheatsheet

This cheatsheet was written to support the `reflective console` article, available at PUT LINK HERE.

It's duplicated here as a standalone reference.



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
