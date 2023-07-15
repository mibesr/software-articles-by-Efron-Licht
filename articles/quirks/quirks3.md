THIS IS UNFINISHED AND UNPUBLISHED.
READ AT YOUR OWN RISK.
# Golang Quirks & Tricks, Pt 2

#### A Programming Article by Efron Licht

#### July 2023

This is a loose sequel to [part 1](https://eblog.fly.dev/quirks.html) and [part 2](https://eblog.fly.dev/quirks2.html) of this series.

## outline

### proxying compile-time-known values



### arrays

#### enum lookup tables

map-like array literals:

Generally speaking, arrays of structs are less efficient than structs of arrays. 

```go
type GunKind int16
const (
    PISTOL GunKind = iota
    RIFLE
    SHOTGUN
    // omitted
    GunKindN // number of gun kinds: must be last
)
var MaxAmmo = [GunKindN]int16{
    PISTOL: 10,
    RIFLE: 30,
    SHOTGUN: 8,
}

var GunNames = [GunKindN]string{
    PISTOL: "pistol",
    RIFLE: "rifle",
    SHOTGUN: "shotgun",
}
```

We can ensure all the values are filled in by using a generic identity function and reflection:



```go
func MustNonZero[T any](a T) T {
    v := reflect.ValueOf(a)
    switch v.Kind() {
        case reflect.Array, reflect.Slice:
            for i := 0; i < v.Len(); i++ {
                if v.Index(i).IsZero() {
                    panic(fmt.Sprintf("array element %d is zero", i))
                }
            }
        default:
            panic("not an array or slice")
    }
    var zero T
    var zeroes 
    for i := range a {
        if a[i] == zero {
            panic(fmt.Sprintf("array element %d is zero", i))
        }
    }
    return a
}
```

### build tags
- overview
- zero-cost 'runtime'    disambiguation