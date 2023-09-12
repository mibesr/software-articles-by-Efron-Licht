# Backend from the Beginning, Part 2: Practical Backend: `net/http` , `context`, `encoding/JSON`

A software article by Efron Licht

September 2023

<<article list placeholder>>

This is the second part of a series of articles about backend development in Go. [In the previous article](https://eblog.fly.dev/backendbasics.html), we covered the basics of TCP, DNS, and HTTP, and wrote a number of simple programs to demonstrate how they work and our own basic HTTP library. In this one, we'll start diving into Go's standard library and  the show how it provides everything you need for basic client/server HTTP communication. We'll also start talking about different serialization formats and designing APIs. We'll finish by discussing the `context` package and some of the unique concerns of web development (timeout, cancellation, etc.).

## The `net/http` package

`net/http` provides a complete HTTP client and server implementation. It's a great place to start when building a web application in Go. Let's start by looking at the client side.

## Requests and Responses

While we built our own `Request` and `Response` types in the previous article, `net/http` provides its own; as Go developers, these are the types you should use 99.9% of the time. Before we move forward, take a peek at the struct definitions of [`http.Request`](https://pkg.go.dev/net/http#Request) and [`http.Response`](https://pkg.go.dev/net/http#Response) structs to see how they differ from ours. The most important difference is that they use `io.Reader`s for the bodies rather than strings, since `HTTP` is a streaming protocol.

### [`http.Request`](https://pkg.go.dev/net/http#Request)

Build a HTTP request with [`http.NewRequestWithContext(ctx, method, url, body)`](https://pkg.go.dev/net/http#NewRequestWithContext). The library will automatically parse the URL and set the `Host` header. The `body` argument is an `io.Reader` that provides the request body. If you don't have a body, pass `nil`.

Never use `http.NewRequest` without a context. If you don't know what context to use, use [`context.TODO()`](https://pkg.go.dev/context#TODO). This will save you from a lot of headaches later on.

The most basic example is a `GET` request with no body:

```go
ctx := context.TODO() // use context.TODO() if you don't know what context to use.
var body io.Reader = nil // nil readers are OK; it means there's no body.
const method = "GET"
const url = "https://eblog.fly.dev/index.html"
req, err := http.NewRequestWithContext(ctx, method, url, body) // the function will parse the URL and set the Host header; invalid URLs will return an error.
```

For a `POST` request, you'll need to provide a body. The simplest way to do this is to use [`strings.NewReader`](https://pkg.go.dev/strings#NewReader) to create a reader from a string:

```go
const method = "POST"
const url = "https://eblog.fly.dev/index.html"
var body io.Reader = strings.NewReader("hello, world")
req, err := http.NewRequestWithContext(ctx, method, url, body)
```

### Headers

Requests automatically set the `Host` header (and a few others, like `User-Agent` and `Accept-Encoding`), but you'll need to set the rest yourself.  Go provides a `Header` type to represent a request or response's HTTP headers. That is, the type represents the complete set of headers for a request or response, not an individual key-value pair.

[`http.Header`](https://pkg.go.dev/net/http#Header) is a `map[string][]string` with some special methods to make it easier to work with. Why `[]string`? Because HTTP allows multiple headers with the same key. For example, the following is a valid HTTP request:

```http
GET / HTTP/1.1
Host: eblog.fly.dev
User-Agent: eblog/1.0
Accept-Encoding: gzip
Accept-Encoding: deflate
Some-Key: somevalue
```

All of `http.Header`'s methods _canonicalize_ keys, turning them into `Title-Case`. For example, `Header.Add("accept-encoding", "gzip")` will add a header with the key `Accept-Encoding`. See the previous article for details on that.

To briefly summarize using `http.Header`:

- use `Header.Add(key, value)` to add a header: it will automatically canonicalize the key to `Title-Case` and append the value to the list of values for that key. Read as `k := AsTitle(key); Header[k] = append(Header[k], value)`.
- use `Header.Set(key, value)` to set a header: it will automatically canonicalize the key to `Title-Case` and set the value for that key to a single-element list containing the value. Read as `Header[AsTitle(key)] = []string{value}`.
- use `Header.Get` to get the _first_ header matching a key, or the empty string if none are found.
- use `Header.Values(key)` to get the list of header values matching the canonical key.

Both `http.Request` and `http.Response` use the same header type.

To demonstrate, let's build the above request.

```go

package main

func main() {
    // https://go.dev/play/p/eE32qPmuDeS
    const method = "GET"
    const url = "https://eblog.fly.dev/index.html"
    var body io.Reader = nil
    req, err := http.NewRequestWithContext(context.TODO(), method, url, body)
    if err != nil {
        log.Fatal(err)
    }
    req.Header.Add("Accept-Encoding", "gzip")
    req.Header.Add("Accept-Encoding", "deflate")
    req.Header.Set("User-Agent", "eblog/1.0")
    req.Header.Set("some-key", "a value")   // will be canonicalized to Some-Key
    req.Header.Set("SOMe-KEY", "somevalue") // will overwrite the above since we used Set rather than Add
    req.Write(os.Stdout)
}

```

`http.Request.Write` serializes the request as HTTP to the provided `io.Writer`. In this case, we're using `os.Stdout`, so it will print the request to the terminal.

We run the program:
IN:

```sh
go run ./main.go
```

OUT:

```http
GET /index.html HTTP/1.1
Host: eblog.fly.dev
User-Agent: eblog/1.0
Accept-Encoding: gzip
Accept-Encoding: deflate
Some-Key: somevalue
```

### Building URLs with [`net/url.Values`](https://pkg.go.dev/net/url#Values)

While you can build a URL by hand, query parameters can occasionally be tricky to get right, since they must be properly escaped. The `url.Values` type provides a convenient way to build query parameters using an API extremely similar to `http.Header`. In the previous article, we searched scryfall for magic cards with the word "ice" in their name, sorted by release date, in ascending order. The URL looked like this:

```go
GET /search?q=ice&order=released&dir=asc HTTP/1.1
Host: scryfall.com
```

This time, let's search for cards with the phrase "of Emrakul" in their name instead. The API documentation mentions we'll need to use double quotes to search for a phrase containing a space; additionally, since it's a URL, we'll need to escape the space between "of" and "Emrakul". This could be tricky to do by hand, so let's use `url.Values`:

Let's build this request using `url.Values`:

IN:

```go
// https://go.dev/play/p/OzX3Ule7Q3r
func main() {
    const method = "GET"
    v := make(url.Values)
    v.Add("q", `"of Emrakul"`) // note we use go's raw string syntax (`) to avoid having to escape the double quotes.
    v.Add("order", "released")
    v.Add("dir", "asc")
    const path = "https://scryfall.com/search"
    dst := path + "?" + v.Encode() // Encode() will escape the values for us. Remember the '?' separator!
    req, err := http.NewRequestWithContext(context.TODO(), method, dst, nil)
    if err != nil {
        log.Fatal(err)
    }
    req.Write(os.Stdout)
}
```

OUT:

```http
GET /search?dir=asc&order=released&q=%22of+Emrakul%22 HTTP/1.1
Host: scryfall.com
User-Agent: Go-http-client/1.1
```

Note that Go automatically added a [`User-Agent`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/User-Agent) header for us alongside the Host.
`net/url` covers much more than just query parameters; it's a complete URL parser and builder. It's not a large package; just take twenty minutes to read the documentation sometime.

### Responses

Responses are broadly similar to requests. I'll cover them as we use them. To briefly summarize:

- Access the response body using `Response.Body`, which is an `io.ReadCloser`.
- Headers are available in `Response.Header`, and the status code is in `Response.StatusCode`.
- See the full HTTP response by calling `Response.Write` with an `io.Writer`.

Only **clients** should see responses; **servers** use the `[http.ResponseWriter](https://pkg.go.dev/net/http#ResponseWriter)` API instead.

## [`http.Client`](https://pkg.go.dev/net/http#Client)

`Client` allows you to make a `Request` to a server using `Do`. The API is simple:

```go
func (c *Client) Do(req *Request) (*Response, error)
```

`Do` is the core method of `http.Client`, and all the other methods are wrappers around it. For the purpose of this article, we will _only_ use `Do`; I suggest you 'do' the same, since it provides a single consistent API.

- `http.Get`, `http.Post`, `http.PostForm`, and `http.Do` are wrappers around `http.DefaultClient.Do`.
- `http.Client.Get`, `http.Client.Post` and `http.Client.PostForm` are also wrappers around `http.Client.Do`. (`PostForm`  is occasionally useful, but the others conceal more than they simplify, IMO).

The following complete program, `download`, uses `http.Client` to download a file from the internet and save it to the local filesystem.

```go
// download is a command-line tool to download a file from a URL.
// usage: download [-timeout duration] url filename
package main

import (
    "context"
    "flag"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

func main() {
    dir := flag.String("dir", ".", "directory to save file")
    timeout := flag.Duration("timeout", 30*time.Second, "timeout for download")
    flag.Parse()
    args := flag.Args()
    if len(args) != 2 {
        log.Fatal("usage: download [-timeout duration] url filename")
    }
    url, filename := args[0], args[1]
    // always set a timeout when you make an HTTP request.
    c := http.Client{Timeout: *timeout}

    // don't worry about the details of context for now; we'll talk about it later in this article.
    // if you don't know what context to use, use context.TODO(). 
    if err := downloadAndSave(context.TODO(), &c, url, filename); err != nil {
        log.Fatal(err)
    }
}
func downloadAndSave(ctx context.Context, c *http.Client, url, dst string) error {
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return fmt.Errorf("creating request: GET %q: %v", url, err)
    }
    resp, err := c.Do(req) // Do serializes a http.Request, sends it to the server, and then deserializes the response to a http.Response. 

    // always check for errors after calling Do. errors from 'Do' usually mean something went wrong on the network.
    if err != nil {
        return fmt.Errorf("request: %v", err)
    }
    defer resp.Body.Close() // always close response bodies when you're done with them.

    // immediately after checking for errors, check the response status code; this is how the server tells us if the request succeeded.
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("response status: %s", resp.Status)
    }

    // ok, we have a successful response. let's save it to a file.

    dstPath := filepath.Join(*dir, filename)
    dstFile, err := os.Create(dstPath)
    if err != nil {
        return fmt.Errorf("creating file: %v", err)
    }
    defer dstFile.Close() // always close files when you're done with them.
    if _, err := io.Copy(dstFile, resp.Body); err != nil { 
        return fmt.Errorf("copying response to file: %v", err)
    }
}
```

Let's try it out by downloading the index page of this blog:
IN:

```sh
go build -o download ./download.go # build the program
./download https://eblog.fly.dev/index.html index.html # save the index page to index.html
cat index.html # print the contents of the file
```

OUTPUT:

```html
<!DOCTYPE html><html><head>
    <title>index.html</title>
    <meta charset="utf-8"/>
    <link rel="stylesheet" type="text/css" href="/dark.css"/>
    </head>
    <body>
    <h1> articles </h1>
<h4><a href="/backendbasics.html">backendbasics.html</a>
</h4><h4><a href="/console.html">console.html</a>
</h4><h4><a href="/mermaid_test.html">mermaid_test.html</a>
</h4><h4><a href="/benchmark_results.html">benchmark_results.html</a>
</h4><h4><a href="/quirks2.html">quirks2.html</a>
</h4><h4><a href="/fastdocker.html">fastdocker.html</a>
</h4><h4><a href="/faststack.html">faststack.html</a>
</h4><h4><a href="/console-autocomplete.html">console-autocomplete.html</a>
</h4><h4><a href="/quirks3.html">quirks3.html</a>
</h4><h4><a href="/cheatsheet.html">cheatsheet.html</a>
</h4><h4><a href="/bytehacking.html">bytehacking.html</a>
</h4><h4><a href="/quirks.html">quirks.html</a>
</h4><h4><a href="/reflect.html">reflect.html</a>
</h4><h4><a href="/startfast.html">startfast.html</a>
</h4><h4><a href="/performanceanxiety.html">performanceanxiety.html</a>
</h4><h4><a href="/article_list.html">article_list.html</a>
</h4><h4><a href="/noframework.html">noframework.html</a>
</h4><h4><a href="/onoff.html">onoff.html</a>
</h4><h4><a href="/README.html">README.html</a>
</h4><h4><a href="/testfast.html">testfast.html</a>
</h4><h4><a href="/index.html">index.html</a>
</h4></body>
```

If we set the timeout to `1ms`, we'll get an error:

IN:

```sh
./download -timeout 1ms https://eblog.fly.dev/index.html index.html
```

OUT:

```text
2023/09/09 09:33:15 request: Get "https://eblog.fly.dev/index.html": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
```

There's that word `context` again: wherever you see timeouts and cancellation, you'll see `context`.

Clients are safe for concurrent use across multiple goroutines. The following complete program, `pardownload` downloads a list of URLs in parallel, saving them to the specified directory; it's a straightforward extension of the previous program.

```go
// pardownload downloads a list of URLs in parallel, saving them to the specified directory.
// It exits with a nonzero status code if any of the downloads fail, where the status code is the number of failed downloads.
package main

import (
    "context"
    "flag"
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "sync"
    "time"
)

func main() {
    var dstDir string
    var client http.Client // the zero value of http.Client is a usable client.
    flag.StringVar(&dstDir, "dst", "", "destination directory; defaults to current directory")
    // set the timeout for the client using a command-line flag.
    flag.DurationVar(&client.Timeout, "timeout", 1*time.Minute, "timeout for the request")
    flag.Parse()

    src := flag.Args()
    if len(src) == 0 {
        log.Fatalf("can't copy")
    }
    dstDir, err := filepath.Abs(dstDir) // make the destination directory absolute, so our error messages are easier to read.
    if err != nil {
        log.Fatalf("invalid destination directory: %v", err)
    }
    dst := make([]string, len(src)) // make a slice of the same length as src, so we can access it in parallel, without worrying about synchronization.
    for i := range src {
        dst[i] = filepath.Join(dstDir, filepath.Base(src[i]))
    }

    errs := make([]error, len(src)) // similarly, make a slice of errors.

    wg := new(sync.WaitGroup) // a WaitGroup waits for a collection of goroutines to finish.
    wg.Add(len(src))          // add the number of goroutines we're going to wait for.

    now := time.Now()
    for i := range src {
        i := i // see https://golang.org/doc/faq#closures_and_goroutines
        go func() {
            defer wg.Done() // tell the WaitGroup that we're done.
            // this is a simple function, so we don't 'really' need to defer, but it's a good habit to get into.
            errs[i] = downloadAndSave(context.TODO(), &client, src[i], dst[i])
        }()
    }
    wg.Wait() // wait for all the goroutines to finish.

    log.Printf("downloaded %d files in %v", len(src), time.Since(now))
    var errCount int // number of errors
    for i := range errs {
        if errs[i] != nil {
            log.Printf("err: %s -> %s: %v", src[i], dst[i], errs[i])
            errCount++
        } else {
            log.Printf("ok: %s -> %s", src[i], dst[i])
        }
    }
    os.Exit(errCount) // nonzero exit codes indicate failure.
}
```

## Contexts and Cancellation

Any internet communication can fail. The network can go down, the server can crash, or the server can just be slow. When I make a network call, I'm implicitly expecting it to finish _soon_, not just 'eventually'; it's no good for me if my request to buy a plane ticket finishes after the flight has already left.

Go's [`context.Context`](https://pkg.go.dev/context#Context)` type is for managing state 'about' a function, rather than 'in' a function. These break down into two large groups: function metadata (start time, request IDs) and deadlines/cancellation. This package is too complex for me to cover in detail here, so I strongly recommend you read both the [package documentation](https://pkg.go.dev/context) and [the blog post that introduced it](https://blog.golang.org/context).

**Use a context to set a ceiling on execution time for any function that does I/O.** I/O includes, but is not limited to, network requests, database queries, and file operations. Many of Go's core packages contain both `context` and `context`-free versions of functions; **always use the `context` version under all circumstances**. If you don't know what context to use, use `context.TODO()`. 

Failure to set timeouts can lead to indefinite resource leaks, denial of service attacks, or hard-to-track-down bugs.

| Function | Replacement | Description | Note |
| --- | --- | --- | --- |
| [`http.NewRequest`](https://pkg.go.dev/net/http#NewRequest) | [`http.NewRequestWithContext`](https://pkg.go.dev/net/http#NewRequestWithContext) | Build an outgoing HTTP request. |
| [`sql.DB.Query`](https://pkg.go.dev/database/sql#DB.Query) | [`sql.DB.QueryContext`](https://pkg.go.dev/database/sql#DB.QueryContext) | Query a database. | |
| [`sql.DB.Exec`](https://pkg.go.dev/database/sql#DB.Exec) | [`sql.DB.ExecContext`](https://pkg.go.dev/database/sql#DB.ExecContext) | Execute a database query. | |
| File.Read / File.Write | N/A | File I/O. | Close the file in a separate goroutine after a timeout instead.|
| [`net.Dial`](https://pkg.go.dev/net#Dial) | [`DialTimeout`](https://pkg.go.dev/net#DialTimeout) / [`net.Dialer.DialContext`](https://pkg.go.dev/net#Dialer.DialContext) | Dialoll a network connection. | |



|Function|Description|Usage|
|---|---|---|
|`context.Background()`|returns a context with no metadata or cancellation signal; effective 'zero value' | To start a new context chain.|
|`context.TODO()`| as `Background()`| During prototyping, when you don't know what context to use.|
|`context.WithCancel(parent)`|returns a context with a cancellation signal that is triggered when `parent` is cancelled| To propagate cancellation signals; usually `WithTimeout` or `WithDeadline` are more appropriate.|
|`context.WithDeadline(parent, deadline)`|returns a context with a cancellation signal that is triggered when `deadline` elapses| To preserve a deadline set by another service.|
|`context.WithTimeout(parent, timeout)`|returns a context with a deadline equal to `time.Now().Add(timeout)`| For any I/O operation.|

The **context should always be the first argument** to any function that that does I/O or could possibly time out. This makes it easy to propagate cancellation signals. For example, the following function uses `context.WithTimeout` to set a timeout for a network request.

Add a context to an outgoing HTTP request using `http.NewRequestWithContext(ctx, method, url, body)`.

Adding a context to _incoming_ requests is a little more complicated.

You have three options, which overlap in functionality:

- The `BaseContext` field of `http.Server` is a context for the _listener_, which is then passed to each request. Note that cancelling this context will cancel _all_ requests based on that listener, so it's generally only appropriate for 'universal' shutdowns, like handling `SIGINT` from the OS. See my article on [turning off software](https://eblog.fly.dev/onoff.html) for more details on signal handling and graceful shutdowns.
- 
- The `ConnContext` field of `http.Server` is the default context for each TCP connection. This is useful for setting a timeout for _all_ requests on a connection, even if they're properly sending packets back and forth.
- You can set the request context manually in a `http.Handler` by wrapping the `http.ResponseWriter` and `*http.Request` in a new `http.Request` with a new context. This is useful for setting a timeout for _individual_ requests, and adding metadata to the context. We'll talk about this one more in the next article when we get into middleware.

It's your job to check the cancellation signal and return an error if it's set. You can check the cancellation signal in two ways: `ctx.Err()` returns the cancellation error; `ctx.Done()` returns a channel that is closed when the context is cancelled.

Either way, you should return an error if the context is cancelled. The following complete program demonstrates how to use `context.WithTimeout` to set a timeout for a network request:

```go


```go

The following program demonstrates how to use `context.WithTimeout` to set a timeout for a network request:

```go
// https://go.dev/play/p/6Z3Z2Z2Z2Z2



```go
// https://go.dev/play/p/6Z3Z2Z2Z2Z2

```go
