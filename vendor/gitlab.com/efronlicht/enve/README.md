# enve

Parsing and configuration via environment variables.

- As simple as I could make it
- Handy error messages
- Lets you know by default what knobs are required or missing.
- Logs at most once per environment variable.

## the big idea

If you have a **required** configuration value, use the **Must** family of functions:

```go
var publicKey = enve.MustString("MY_APP_PUBLIC_KEY")
```

If you have a **sensible default**, use the **Or** family of functions:

```go
// if MY_APP_PORT is set, use that, but otherwise, 8080
var port = enve.IntOr("MY_APP_PORT", 8080)
```

You can use custom parsing functions easily with MustParse or ParseOr:

```go
var requiredToken []byte = enve.MustParse(base64.StdEncoding.DecodeString, "REQUIRED_TOKEN_THATS_IN_BASE64_FOR_SOME_REASON")
```

```go
type Point3 struct {X, Y, Z float64}
func pointFromString(s string)(p Point3, err error) {
    _, err := fmt.Sscan(s, &p.X, &p.Y, &p.Z)
    return p, err
}
var startingPoint = enve.ParseOr(pointFromString, "STARTING_POINT", Point3{0, 0, 0})
```

### demonstrated example

in [example.go](./example/example.go)  

```go

func main() {
 port := enve.IntOr("APP_PORT", 8080)
 name := enve.MustString("FRIENDLY_NAME")
 _ = enve.DurationOr("TIMEOUT", 1*time.Minute)
 _ = enve.MustInt("SOME_INT")
 http.ListenAndServe(fmt.Sprintf(":%d", port), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
  fmt.Fprintf(w, "hello, %q", name)
 }))
}
```

in

```sh
 go run ./example
```

out

```
2023/01/04 17:12:58 enve: logging enabled. disable enve logging with environment variable ENVE_LOGDISABLED=1
2023/01/04 17:12:58 enve: missing optional envvar APP_PORT: falling back to default: 8080; caller main.main (/home/efron/go/src/gitlab.com/efronlicht/enve/example/example.go 12)
2023/01/04 17:12:58 enve: FATAL ERR: missing required envvar FRIENDLY_NAME; caller main.main (/home/efron/go/src/gitlab.com/efronlicht/enve/example/example.go 13)
panic: missing required envvar FRIENDLY_NAME
```

in

```sh
FRIENDLY_NAME=efron SOME_INT=NaN go run ./example/
```

out

```
2023/01/04 17:11:22 enve: logging enabled. disable enve logging with environment variable ENVE_LOGDISABLED=1
2023/01/04 17:11:22 enve: missing optional envvar APP_PORT: falling back to default: 8080; caller main.main (/home/efron/go/src/gitlab.com/efronlicht/enve/example/example.go 12)
2023/01/04 17:11:22 enve: invalid optional envvar TIMEOUT: time: invalid duration "notatimeout": falling back to default  1m0s; caller main.main (/home/efron/go/src/gitlab.com/efronlicht/enve/example/example.go 14); parser time.ParseDuration (/usr/local/go/src/time/format.go 1522)
2023/01/04 17:11:22 enve: FATAL ERR: parsing required envvar SOME_INT into type int: strconv.Atoi: parsing "NaN": invalid syntax; caller main.main (/home/efron/go/src/gitlab.com/efronlicht/enve/example/example.go 15); parser strconv.Atoi (/usr/local/go/src/strconv/atoi.go 231):
panic: strconv.Atoi: parsing "NaN": invalid syntax
```
