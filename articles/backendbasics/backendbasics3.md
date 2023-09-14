# backend basics, pt 3: Middleware & Routing

A software article by Efron Licht.

September 2023

<<article list placeholder>>

This is the third article in a series on backend development in Go that aims to teach **understanding** of how backend is put together, rather than just lego-like assembly of pre-built components. 

In the [first article](https://eblog.fly.dev/backendbasics.html), we went through the components of the internet up to HTTP - TCP, IP, & DNS, and built our own HTTP library and basic servers. 

In the [second article](https://eblog.fly.dev/backendbasics2.html), we went on a tour through the standard library's networking packages and demonstrated how to build HTTP servers and clients in practice, rather than theory.

We'll now cover how to add sophisticated behavior to our servers and clients like authentication, logging, and tracing using **middleware**. We'll also cover the 'missing component' of Go's networking stack: **routing**.

In the fourth article, we'll briefly cover **testing**, **databases**, and **dependency injection**.

Then we'll finish up with a fifth and final opinion piece talking about why it's good to build things yourself.


## middleware

### client middleware

There's a lot of behavior we might want to add to many of our outgoing HTTP requests.

 A few things that come to mind off the top of my head:

- Timing how long it takes for a request to complete
- Adding an authorization header
- Retrying failed requests with exponential backoff
- Collecting metrics on the number of requests made to each endpoint
- Logging failed requests
- Set `Accept-Encoding: gzip` header on requests to let the server know we can handle gzipped responses
- Transparently unzip responses sent with the `'Content-Encoding: gzip'` header (actually, go's http client does this for us already, but sshhh)
- etc, etc, etc

The total weight of this code is probably substantial. If we added all of these behaviors to simple routes that just make a GET request and unmarshal the reponse to JSON, our 'wrapper' code would quickly dwarf the actual business logic.

Another approach might be to make our own `DoRequest` function that would encapsulate all this behavior. This is certainly possible, though it gets rather complex. Here's what that might look like for a subset of the above behaviors:

```go

// DoRequest is a helper function that sends the given request using the given client. It adds the following functionality:
//   - adds a context to the request
//   - adds an authorization header to the request
//   - retries the request up to 3 times if the server is unavailable or returns a 5xx status code
//   - returns an error if the server returns a 4xx status code
//   - logs the request duration
//   
func DoRequest(ctx context.Context, c *http.Client, r *http.Request) (*http.Response, error) {
    r = r.WithContext(ctx) // add context to request
    // track execution time
    start := time.Now()
    defer func() { log.Printf("request took %s", time.Since(start)) }()

    r = addAuthHeader(r) // add auth header to request

    // retry logic
    var retryErrs error
    for retry := uint(0); retry < 3; retry++ {
        if retry > 0 {
            time.Sleep(10 * time.Millisecond << retry)
        }
        resp, err := c.Do(r)
        if errors.Is(retryErrs, syscall.ECONNREFUSED) || errors.Is(retryErrs, syscall.ECONNRESET) {
            retryErrs = errors.Join(retryErrs, err)
            continue
        }
        if retryErrs != nil {
            return nil, fmt.Errorf("failed after %d retries: %w", retry, retryErrs)
        }
        switch sc := resp.StatusCode; {
        case sc <= 200 && sc < 400:
            return resp, nil // success! we're done here.
        case sc <= 400 && sc < 500: // 4xx status code
            return nil, fmt.Errorf("failed after %d retries: %s", retry, resp.Status)
        default: // 5xx, 1xx, or unknown status code
            retryErrs = errors.Join(retryErrs, fmt.Errorf("try %d: %s", retry, resp.Status))
        }

    }
    return nil, fmt.Errorf("failed after 3 retries: %w", retryErrs)

}
```

Then we could simply replace `client.Do` with `DoRequest(client, r)`.

This has some advantages:

- Only one place to look
- Simple control flow
- Easy to add new functionality

But things get difficult very quickly if we want to be able to add _some_ but not all of this functionality to a request. For example:

- Different routes might need different authorization headers (what if we're hitting two different services?)
- One route might need a longer timeout b/c it's known to be slow.
- Some routes must be rate limited to avoid overloading the server, but others are O.K. to hit as hard as we can.

What we really need is some kind of composability, where we can quickly apply _some_ of the options to a client on an as-needed basis.

This is where **middleware** comes in. Essentially, middleware "wraps" a client, sitting between it and the outside world. It modifies requests and responses as they pass through it, and can short-circuit the request/response cycle entirely.

For **clients**, we build middleware by wrapping the Client's `http.RoundTripper` interface. Usually, a Client uses the `http.DefaultTransport`, which is an `http.Transport` that handles all the low-level details of making an HTTP request. We can wrap this transport with our own middleware, and then pass it to the Client.

A brief note: the [documentation for RoundTripper](https://pkg.go.dev/net/http#RoundTripper) says that it shouldn't modify the request or response. I disagree with this; it's simpler and easier to intercept the RoundTripper than to build another layer _on top_ of **http.Client**. Over the years, this seems to be the consensus for backend development in Go. If you're not comfortable with this, you can build a layer on top of the Client, but we'll be using the RoundTripper approach.

The `RoundTripper` interface looks like this:

```go
type RoundTripper interface {
    RoundTrip(*http.Request) (*http.Response, error)
}
```

### Building Client Middleware

We'll follow the example of the `http.HandlerFunc` and build a `RoundTripFunc` type that implements this interface:

```go

// RoundTripFunc is an adapter to allow the use of ordinary functions as RoundTrippers, a-la http.HandlerFunc
type RoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface by calling f(r)
func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {return f(r)}

var _ http.RoundTripper = RoundTripFunc(nil) // assert that RoundTripFunc implements http.RoundTripper at compile time
```

This allows us to use closures as simple roundtrippers. Let's build a package, `clientmw`, and implement a few simple middlewares.

Each middleware will be a function that takes a `http.RoundTripper` and returns a `RoundTripFunc` that wraps it.

The most basic kind of middleware has no other arguments, and simply wraps the `RoundTripper` in a closure. Here's an example:

```go
package clientmw

// we'll use this helper function to log the beginning and end of each middleware. no need for this in the real world,
// but it should help you understand what's going on.
func logExec(name string) func() {
 log.Printf("middleware: begin %s", name)
 return func() { defer log.Printf("middleware: end %s", name) }
}
// TimeRequest returns a RoundTripFunc that logs the duration of the request.
func TimeRequest(rt http.RoundTripper) RoundTripFunc {
    return func(r *http.Request) (*http.Response, error) {
        // for demonstration purposes, we'll add these logs to each middleware; don't do this in production!
        defer logExec("TimeRequest")()

        start := time.Now()
        resp, err := rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
        if err != nil {
            log.Printf("%s %s: errored after %s", r.Method, r.URL, time.Since(start))
            return nil, err
        }
        log.Printf("%s %s: %d %s in %s", r.Method, r.URL, resp.StatusCode, http.StatusText(resp.StatusCode), time.Since(start))
        return resp, nil
    }
}
```

But generally speaking you'll want to pass some arguments to your middleware. For example, we might want to have configurable retries:

```go
package clientmw

// RetryOn5xx returns a RoundTripFunc that retries the request up to n times if the server returns a 5xx status code.
// It will use exponential backoff: first retry will be after wait, second after 2*wait, third after 4*wait, etc.
func RetryOn5xx(rt http.RoundTripper, wait time.Duration, tries int) RoundTripFunc {
    // validate arguments OUTSIDE of the closure, so that it only happens once
    if n <= 1 {
        panic("n must be > 1")
    }
    if wait <= 0 {
        panic("wait must be > 0")
    }
    return func(r *http.Request) (*http.Response, error) {
        defer logExec("RetryOn5xx")()
            // retry logic
        var retryErrs error
        for retry := uint(0); retry < tries; retry++ {
            if retry > 0 {
                time.Sleep(wait << retry)
            }
            resp, err := rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
            if errors.Is(retryErrs, syscall.ECONNREFUSED) || errors.Is(retryErrs, syscall.ECONNRESET) {
                retryErrs = errors.Join(retryErrs, err)
                continue
            }
            if retryErrs != nil {
                return nil, fmt.Errorf("failed after %d retries: %w", retry, retryErrs)
            }
            switch sc := resp.StatusCode; {
            case sc <= 200 && sc < 400:
                return resp, nil // success! we're done here.
            case sc <= 400 && sc < 500: // 4xx status code
                return nil, fmt.Errorf("failed after %d retries: %s", retry, resp.Status)
            default: // 5xx, 1xx, or unknown status code
                retryErrs = errors.Join(retryErrs, fmt.Errorf("try %d: %s", retry, resp.Status))
            }

        }
        return nil, fmt.Errorf("failed after 3 retries: %w", retryErrs)
    }
}
```

Most middleware modifies the `context` of the request; this allows later middlewares to access the values set by earlier ones. For example, we may wish to **trace** our requests, adding a unique ID to each request that will be be associated with every log and carried from service to service via the headers. We can do this with a middleware that keeps track of a `Trace` struct in the context.

Additionally, we'll use the [`github.com/google/uuid`](https://pkg.go.dev/github.com/google/uuid) package to generate unique IDs. I talk about uuids in some detail in my article on [simple byte hacking](https://eblog.fly.dev/bytehacking.html); don't worry about it for now.

```go
import "github.com/google/uuid"
type Trace struct {
    TraceID uuid.UUID // TraceID is unique across the lifecycle of a single 'event', regardless of how many requests it takes to complete. Carried in the `X-Trace-ID` header.
    RequestID uuid.UUID // RequestID is unique to each request. Carried in the `X-Request-ID` header.
}
```

We'll use the following generic methods to add and retrieve values of a type from a context. See [my article on Go quirks and tricks, pt 1](https://eblog.fly.dev/quirks.html) for more details on how this works.

```go
package ctxutil
type key[T any] struct{} // key is a unique type that we can use as a key in a context

// WithValue returns a new context with the given value set. Only one value of each type can be set in a context; setting a value of the same type will overwrite the previous value.
func WithValue[T any](ctx context.Context, value T) context.Context {
    return context.WithValue(ctx, key[T]{}, value)
}
// Value returns the value of type T in the given context, or false if the context does not contain a value of type T.
func Value[T any](ctx context.Context) (T, bool) {
    value, ok := ctx.Value(key[T]{}).(T)
    return value, ok
}
```

```go
package trace

import (
    "github.com/google/uuid"
)

type Trace struct {
    TraceID uuid.UUID // TraceID is unique across the lifecycle of a single 'event', regardless of how many requests it takes to complete. Carried in the `X-Trace-ID` header.
    RequestID uuid.UUID // RequestID is unique to each request. Carried in the `X-Request-ID` header.
}
```

```go
package clientmw
func Trace(rt http.RoundTripper) http.RoundTripper {
    return func(r *http.Request) (*http.Response, error) {
        defer logExec("Trace")()
        traceID, err := uuid.Parse(r.Header.Get("X-Trace-ID"))
        if err != nil {
            traceID = uuid.New()
        }

        // build the trace. it's a small struct, so we put it directly in the context and don't bother with a pointer
        trace := trace.Trace{ TraceID: traceID, RequestID: uuid.New()}
        

        ctx := ctxutil.WithValue(r.Context(), trace) // add trace to context; retrieve with ctxutil.Value[Trace](ctx)
        r = r.WithContext(ctx) // add context to request

        // add trace id & request id to headers
        r.Header.Set("X-Trace-ID", trace.TraceID.String())  
        r.Header.Set("X-Request-ID", trace.RequestID.String())
        return rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
    }
}
```

Let's pick up this trace in the next middleware, one that adds a logger to our requests. We'll just use the standard library `log` package for now, but in practice you should probably use a structured logger; the standard library has recently introduced the `log/slog` package for this purpose.

This will supersede our original `TimeRequest` middleware, so we'll add the timing logic here as well.

```go
package clientmw
// Log returns a RoundTripFunc that logs the request duration and status code. It uses the trace from the context as a prefix, if it exists. See Trace in this package and servermw.Log for the server-side implementation.
func Log(rt http.RoundTripper) RoundTripFunc {
    return func(r *http.Request) (*http.Response, error) {
        defer logExec("Log")()
        trace, ok := ctxutil.Value[Trace](r.Context())
        if ok {
            prefix := fmt.Sprintf("%s %s: [%s %s]: ", r.Method, r.URL, trace.TraceID, trace.RequestID)
        } else {
            prefix := fmt.Sprintf("%s %s: ", r.Method, r.URL)
        }

        logger := log.New(os.Stderr, prefix, log.LstdFlags | log.Lshortfile)
        ctx := ctxutil.WithValue(r.Context(), logger) // add logger to context; retrieve with ctxutil.Value[log.Logger](ctx)
        r = r.WithContext(ctx) // add context to request

        start := time.Now()
        resp, err := rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
        if err != nil {
            logger.Printf("errored after %s: %s", time.Since(start), err)
            return nil, err
        }
        logger.Printf("%d %s in %s", resp.StatusCode, http.StatusText(resp.StatusCode), time.Since(start))
        return resp, nil
    }
}
```

### Using Client Middleware

Using our middleware is simple. We just wrap the `http.DefaultTransport` with our middleware, and use it to build a new `http.Client`. It's important to note that middleware is applied "first-in, last-out"; that is, the first middleware we apply will be the last one to run, and the last middleware we apply will be the first one to run!

```go
func clientMiddleware() http.RoundTripper {
    var rt RoundTripFunc // specify the type as a RoundTripFunc, not a http.RoundTripper, so that we don't have to repeatedly wrap it in RoundTripFunc(rt)
    const wait, tries = 10 * time.Millisecond, 3
    // first middleware applied will be the last one to run.
    rt = clientmw.RetryOn5xx(http.DefaultTransport, wait, tries) // retry on 5xx status codes
    rt = clientmw.Log(rt) // log request duration and status code; uses trace from next middleware
    rt = clientmw.Trace(rt) // add trace id to request header
    return rt
}
```

Let's test this out. The following full program, `clientmiddlewareex` makes a GET request to the specified URL, and prints the response body to stdout, using our middleware.

```go
// clientmiddlewareex makes a GET request to the specified URL, and prints the response body to stdout, using our middleware.
package main

import (
    "context"
    "io"
    "log"
    "net/http"
    "os"
    "time"

    "gitlab.com/efronlicht/blog/articles/backendbasics/cmd/clientmiddlewareex/clientmw"
)


func main() {
    if len(os.Args) < 2 {
        log.Fatal("target url required")
    }
    target := os.Args[1]
    client := &http.Client{Transport: clientMiddleware(), Timeout: 5 * time.Second}
    req, err := http.NewRequestWithContext(context.TODO(), "GET", target, nil)
    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Body.Close()
    io.Copy(os.Stdout, resp.Body)
}
```

IN:

```bash
go run clientmiddlewareex.go https://eblog.fly.dev
```

OUT:

```text
2023/09/12 06:43:53 middleware: begin trace
2023/09/12 06:43:53 middleware: begin log
2023/09/12 06:43:53 middleware: begin retryOn5xx
2023/09/12 06:43:53 middleware: end retryOn5xx
GET https://eblog.fly.dev/index.html: [8c63dffb-2901-4ebc-bd7c-73ea843f89e2 9a56e7e8-062f-42db-b087-7018cd6a3610]: 2023/09/12 06:43:53 clientmw.go:103: 200 OK in 88.321876ms
2023/09/12 06:43:53 middleware: end log
2023/09/12 06:43:53 middleware: end trace
```

Checking my server logs, I note that the request was received with the following headers:

```text
X-Request-Id: 9a56e7e8-062f-42db-b087-7018cd6a3610
X-Trace-Id: 8c63dffb-2901-4ebc-bd7c-73ea843f89e2
```

Looks like everything is working as expected. Let's move on to **server middleware**.

### Server Middleware

Server middleware is very similar to Client middleware. Rather than wrapping a `RoundTripper`, we wrap a `http.Handler`. The [`http.Handler`](https://pkg.go.dev/net/http#Handler) interface looks like this:

```go
type Handler interface {
    ServeHTTP(http.ResponseWriter, *http.Request)
}
```

We don't need to define our own `HandlerFunc` type, because [the standard library already provides one](https://pkg.go.dev/net/http#HandlerFunc):

```go
type HandlerFunc func(http.ResponseWriter, *http.Request)
```

We can use this to build our own middleware by wrapping handlers in a closure and returning a `HandlerFunc`, just like we wrapped `RoundTripper`s in a closure and returned a `RoundTripFunc`.

Let's add traces and logs to our server. This is broadly symmetrical to the client middleware:

```go

package servermw

import (
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/google/uuid"
    "gitlab.com/efronlicht/blog/articles/backendbasics/cmd/ctxutil"
    "gitlab.com/efronlicht/blog/articles/backendbasics/cmd/trace"
)

// Trace returns a middleware that injects a trace into the request context,
// picking up the trace id from the request header if it exists, or generating a new one if it doesn't.
// See clientmw.Trace for the client-side implementation.
func Trace(h http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        // get trace/req id from request header, or generate new ones if they don't exist
        traceID, err := uuid.Parse(r.Header.Get("X-Trace-Id"))
        if err != nil {
            traceID = uuid.New()
        }
        reqID, err := uuid.Parse(r.Header.Get("X-Request-Id"))
        if err != nil {
            reqID = uuid.New()
        }

        // pop trace into context, and pop context into request
        trace := trace.Trace{TraceID: traceID, RequestID: reqID}
        ctx = ctxutil.WithValue(ctx, trace)
        r = r.WithContext(ctx)

        // serve the request using the populated context
        h.ServeHTTP(w, r)
    }

}

// Log returns a middleware that injects a logger into the request context. See clientmw.Log for the client-side implementation.
//  It uses the trace from the context as a prefix, if it exists. For most servers, use a structured logger instead; that API is outside the scope of this article.
func Log(h http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        trace, ok := ctxutil.Value[trace.Trace](r.Context())
        var prefix string
        if ok {
            // like GET /articles: [trace-id request-id]:
            prefix = fmt.Sprintf("%s %s: [%s %s]: ", r.Method, r.URL, trace.TraceID, trace.RequestID)
        } else {
            // like GET /articles:
            prefix = fmt.Sprintf("%s %s: ", r.Method, r.URL)
        }
        logger := log.New(os.Stderr, prefix, log.LstdFlags)
        ctx := ctxutil.WithValue(r.Context(), logger)
        r = r.Clone(ctx)
        h.ServeHTTP(w, r)
    }
}
```

Some server middleware may want to track or intercept writes to the response headers or body. Let's list a few examples:

- Automatically gzip-encode writes to the body if the client sent an ``Accept-Encoding: gzip` header.
- Track the status code of the response so we can add it to our logs or metrics.
- Track the total number of bytes written to the response body so we can add it to our metrics.
- Rewrite certain headers so as not to leak internal information to the client.

We can do this by wrapping the `ResponseWriter` in a custom struct that implements the `http.ResponseWriter` interface. This is a bit more complex than wrapping a `RoundTripper` or `Handler`. It's easiest to demonstrate with an example.

The following middleware, `RecordResponse`  and it's associated `RecordingResponseWriter` struct will track the status code and bytes written to the response body, and log them when the request is complete.

```go
package servermw

// RecordResponse returns a middleware that records the response status code and total bytes written to the response.
func RecordResponse(h http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        rrw := &RecordingResponseWriter{RW: w}
        start := time.Now()
        h.ServeHTTP(rrw, r)
        elapsed := time.Since(start)
        // use the logger from the context if it exists
        logger, ok := ctxutil.Value[*log.Logger](r.Context())
        if !ok {
            // fall back to the default logger
            log.Printf("%s %s: %d %s: %d bytes in %s", r.Method, r.URL, rrw.StatusCode, http.StatusText(rrw.StatusCode), rrw.Bytes, elapsed)
            return
        }
        logger.Printf("%d %s: %d bytes in %s", rrw.StatusCode, http.StatusText(rrw.StatusCode), rrw.Bytes, elapsed)
    }
}

// RecordingResponseWriter is an http.ResponseWriter that keeps track of the status code and total body bytes written to it.
type RecordingResponseWriter struct {
    // underlying response writer
    RW         http.ResponseWriter
    StatusCode int // first status code written to the response writer
    Bytes      int // total bytes written
}

// WriteHeader sets the status code, if it hasn't been set already.
func (w *RecordingResponseWriter) WriteHeader(statusCode int) {
    if w.StatusCode == 0 { // first status code written; track it
        w.StatusCode = statusCode
    }
    w.RW.WriteHeader(statusCode) // write to underlying response writer
}

// Header just returns the underlying response writer's header.
func (w *RecordingResponseWriter) Header() http.Header { return w.RW.Header() }

// Write writes the given bytes to the underlying response writer, setting the status code to 200 if it hasn't been set already.
func (w *RecordingResponseWriter) Write(b []byte) (int, error) {
    if w.StatusCode == 0 {
        w.WriteHeader(http.StatusOK)
    }
    n, err := w.RW.Write(b) // write to underlying response writer
    w.Bytes += n            // update total bytes written
    return n, err
}
```

Let's add one last server middleware, `Recovery`, to protect our server from unexpected panics. While ideally we would write perfect code without panics, everyone makes mistakes, and it would be good to be able to continue _some_ service even if one of our endpoints panics under certain conditions.

As before, our Recovery handle takes advantage of the log injected into the context (if it exists). **It's good to have a 'fallback' for any context value you use, since context values are not visible in the type signature or guaranteed to exist.**

```go
// Recovery returns a middleware that recovers from panics, writing a 500 status code and "internal server error" message to the response,
// and logging the panic and associated stack trace.
func Recovery(h http.Handler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        defer func() { // recover from panic
            if err := recover(); err != nil { // recover from panic
                stack := debug.Stack()
                logger, ok := ctxutil.Value[*log.Logger](r.Context())
                if !ok { // use the default logger
                    log.Printf("%s %s: panic: %v\n%s", r.Method, r.URL, err, stack)
                } else { // use the logger from the context
                    logger.Printf("panic: %v\n%s", err, stack)
                }
                // write 500 status code and "internal server error" message to response so it doesn't hang
                w.WriteHeader(http.StatusInternalServerError)
                _, _ = w.Write([]byte("internal server error"))
            }
        }()
        h.ServeHTTP(w, r)
    }
}
```

### Using Server Middleware

The following complete program, `servermiddlewareex`, implements a simple server that serves two endpoints. `GET /time` returns the current time in RFC3339 format, and `GET /panic` panics. Any other endpoint returns a 404.

```go
package main

import (
    "errors"
    "flag"
    "fmt"
    "log"
    "net/http"
    "time"

    "gitlab.com/efronlicht/blog/articles/backendbasics/cmd/servermw"
)

func main() {
    port := flag.Int("port", 8080, "port to listen on")
    flag.Parse()
    // our base handler.
    var h http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
        // route the request. note that there's no need for ANY router, even the stdlib's http.ServeMux
        // if you have a simple enough routing scheme.
        // a switch statement is perfectly fine.
        switch r.URL.Path {
        case "/time":
            fmt.Fprintln(w, time.Now().Format(time.RFC3339))
        case "/panic":
            panic("oh my god JC, a bomb!")
        default:
            http.NotFound(w, r)
        }
    }
    // remember, middleware is applied in First In, Last Out order.

    h = servermw.RecordResponse(h)
    h = servermw.Recovery(h)
    h = servermw.Log(h)
    h = servermw.Trace(h)

    // always apply timeouts to your server, even if you've put cancellations in the context using a middleware.
    server := http.Server{
        Addr:              fmt.Sprintf(":%d", *port),
        Handler:           h,
        ReadTimeout:       1 * time.Second,
        WriteTimeout:      1 * time.Second,
        ReadHeaderTimeout: 200 * time.Millisecond,
    }
    log.Printf("listening on %s", server.Addr)
    if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
        log.Fatal(err)
    }
}
```

Let's fire it up and visit `http://localhost:8080/time`, `http://localhost:8080/panic`, and `http://localhost:8080/foobar` to see what happens.
IN:

```
go run servermiddlewareex.go
```

OUT (client):

```text
localhost:8080/time: 2023-09-12T07:55:16-07:00
localhost:8080/panic: internal server error
localhost:8080/foobar: 404 page not found
```

OUT (server)

```text
2023/09/12 07:50:39 listening on :8080
GET /time: [4c5ef2f9-cd58-4dec-a28f-770b5786fcba fcc3b01b-d18d-49a8-96b2-0514c7ac24c6]: 2023/09/12 07:55:16 200 OK: 26 bytes in 3.636µs
GET /panic: [98f8f8cc-18ce-4b18-9044-3258c24e57e1 88b97381-1866-4689-a23a-fac50bea0da0]: 2023/09/12 07:55:31 panic: oh my god JC, a bomb!
goroutine 6 [running]:
runtime/debug.Stack()
        /usr/local/go/src/runtime/debug/stack.go:24 +0x5e
<snip... stack trace ...>
```

Looks like everything is working as expected. Let's talk about routing.

### Routing

**Routing** is the process of matching a request to a handler via it's `METHOD` and `PATH`. In Go, there's nothing particularly special about routing: it's just something that the `Handler` inside your `Server` does.

The most basic kind of routing is just a `switch` statement, like we saw above. That only dealt with paths, but routing based off `METHOD` is just as easy: the following code is the routing that serves **the website you're reading this on**.

```go
var router = http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
            p := strings.TrimSuffix(r.URL.Path, "/")
            switch {
            case r.Method != "GET": // only GET is allowed
                w.WriteHeader(http.StatusMethodNotAllowed)
                return
            case p == "/debug/uptime": // return uptime
                d := (time.Since(start).Seconds())
                const MIN = 60
                const HOUR = 60 * MIN
                const DAY = 24 * HOUR
                _, _ = fmt.Fprintf(w, "%2dd %02dh %02dm %02ds", int(d/DAY), int(d/HOUR)%24, int(d/MIN)%60, int(d)%60)
                return
            case p == "/debug/meta": // return metadata about the server
                _, _ = w.Write(metaJSON)
                return
            case p == "": // redirect to index.html
                http.Redirect(w, r, "./index.html", http.StatusPermanentRedirect)
                return
            default: // serve webpages
                // fonts are immutable and large, so we can cache them for a long time.~
                // everything else is tiny and might change, so we don't cache it.
                if strings.Contains(r.URL.Path, ".woff2") {
                    r.Header.Add("cache-control", "immutable")
                    r.Header.Add("cache-control", "max-age=604800")
                    r.Header.Add("cache-control", "public")
                } else {
                    r.Header.Add("cache-control", "no-cache")
                }
                static.ServeFile(w, r)
                return
            }
}
```

**This is all you need for small programs.**. For convenience, Go's stdlib comes with a built-in Router, `[http.ServeMux](https://pkg.go.dev/net/http#ServeMux)`, which uses a simple prefix-based matching scheme: the longest prefix that matches the request path wins. It's implemented like this:

```go
// Find a handler on a handler map given a path string.
// Most-specific (longest) pattern wins.
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
    // Check for exact match first.
    v, ok := mux.m[path]
    if ok {
        return v.h, v.pattern
    }

    // Check for longest valid match.  mux.es contains all patterns
    // that end in / sorted from longest to shortest.
    for _, e := range mux.es {
        if strings.HasPrefix(path, e.pattern) {
            return e.h, e.pattern
        }
    }
    return nil, ""
}
```

ServeMux is perfectly fine for the vast majority of backend programs, but is not very flexible.

More complicated routers, like [gorilla/mux](https://github.com/gorilla/mux), allow for routing by path, by patterns matching regular expressions, and for extracting variables from the URL path. I'll quote their documentation here to give you an idea of what this looks like:

- ## HTTP Servemux: quoted documentation

    Let's start registering a couple of URL paths and handlers:

    ```go
    func main() {
        r := mux.NewRouter()
        r.HandleFunc("/", HomeHandler)
        r.HandleFunc("/products", ProductsHandler)
        r.HandleFunc("/articles", ArticlesHandler)
        http.Handle("/", r)
    }
    ```

    Here we register three routes mapping URL paths to handlers. This is equivalent to how `http.HandleFunc()` works: if an incoming request URL matches one of the paths, the corresponding handler is called passing (`http.ResponseWriter`, `*http.Request`) as parameters.

    Paths can have variables. They are defined using the format `{name}` or `{name:pattern}`. If a regular expression pattern is not defined, the matched variable will be anything until the next slash. For example:

    ```go
    r := mux.NewRouter()
    r.HandleFunc("/products/{key}", ProductHandler)
    r.HandleFunc("/articles/{category}/", ArticlesCategoryHandler)
    r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)
    ```

    The names are used to create a map of route variables which can be retrieved calling `mux.Vars()`:

    ```go
    func ArticlesCategoryHandler(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "Category: %v\n", vars["category"])
    }
    ```

    Paths can have variables. They are defined using the format `{name}` or `{name:pattern}`. If a regular expression pattern is not defined, the matched variable will be anything until the next slash. For example:

    ```go
    r := mux.NewRouter()
    r.HandleFunc("/products/{key}", ProductHandler)
    r.HandleFunc("/articles/{category}/", ArticlesCategoryHandler)
    r.HandleFunc("/articles/{category}/{id:[0-9]+}", ArticleHandler)
    ```

    The names are used to create a map of route variables which can be retrieved calling `mux.Vars()`:

    ```go
    func ArticlesCategoryHandler(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        w.WriteHeader(http.StatusOK)
        fmt.Fprintf(w, "Category: %v\n", vars["category"])
    }
    ```

Let's build our own router that can handle expressions of this kind.

```go
// route matches a path to a handler, and extracts path parameters from the path.

type Router struct {
    routes []struct {
        raw string // the raw pattern string
        pattern *regexp.Regexp
        names []string
        methods []string // the methods this route matches; if empty, matches all methods
        handler http.Handler
    }
}
// Vars is a map of path parameters to their values.
type PathVars map[string]string

func buildRoute(pattern string) (pattern *regexp.Regexp, names []string, err error) {
    if pattern == "" || pattern[0] != '/' {
        return nil, nil, fmt.Errorf("invalid pattern %s: must begin with '/'", pattern)
    }
    fields := strings.Split(pattern, "/") // split the pattern into '/'-separated fields
    var rawPattern strings.Builder
    for i, f := range fields {
        w.WriteByte('/')
        if len(f) > 2 && f[0] == '{' && f[1] == '}' { // path parameter
            if i == 0 {
                return nil, nil, fmt.Errorf("path parameter %s cannot be the first field", f)
            }
            name := f[1:len(f)-1] // strip off the '{' and '}'
            names = append(names, name)
            if before, after, ok := strings.Cut(name, ':'); ok { // its a regexp-capture group
                w.WriteByte('(')
                w.WriteString(after)
                w.WriteByte(')')
            } else {
                w.WriteString(`([^/]+)`) // capture anything but a '/'
            }
        } else {
            w.WriteString(f) // literal string.
        }
        
    }
    // check for duplicate path parameters
    for i := range names {
        for j := i; j < len(names); j++ {
            if names[i] == names[j] {
                return route{}, fmt.Errorf("duplicate path parameter %s", names[i])
            }
        }
    }
    pattern, err := regexp.Compile(rawPattern.String())
    if err != nil {
        return route{}, fmt.Errorf("invalid regexp %s: %w", rawPattern.String(), err)
    }
    return pattern, names, nil
}

func (r *Router) AddRoute(rawPattern string, h http.Handler, methods ...string) error {
    pattern, names, err := buildRoute(pattern)
    if err != nil {
        return err
    }
    if len(methods) == 0 {
        methods = nil
    }
    r.routes = append(r.routes, struct {
        pattern *regexp.Regexp
        names []string
        handler http.Handler
        raw string
    }{pattern: pattern, names: names, handler: h, raw: rawPattern, methods: methods})
    sort.Slice(r.routes, func(i, j int) bool { 
        return len(r.routes[i].raw) > len(r.routes[j].raw) || (len(r.routes[i].raw) == len(r.routes[j].raw) && r.routes[i].raw < r.routes[j].raw) // sort by length, then lexicographically
    })
    return nil
}
func (r *Router) MustAddRoute(pattern string, h http.Handler, methods ...string) {
    if err := r.AddRoute(pattern, h); err != nil {
        panic(err)
    }
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
    for _, route := range r.routes {
        if !route.pattern.MatchString(r.URL.Path) {
            continue
        }
        if len(route.methods) == 0 { // all methods match; skip the check.
            goto METHOD_OK // goto considered awesome. 
        }
        // only the specified methods match; check if the request method is one of them.
        for _, method := range route.methods {
            if method == r.Method {
                goto METHOD_OK 
            }
        }
        continue // no method match; maybe the next route will match?

        METHOD_OK:
        matches := route.pattern.FindStringSubmatch(req.URL.Path)
        if len(matches) != len(route.names) + 1 { // +1 because the first match is the entire string
            panic("programmer error: matched regexp has wrong number of matches")
        }
        vars := make(PathVars, len(route.names)) 
        for i, val := range matches[1:] { // again, skip the first match, which is the entire string
            vars[route.names[i]] = match
        }
        ctx := ctxutil.WithValue(req.Context(), vars) // add path vars to context to be retrieved by the handler using ctxutil.Value[PathVars](ctx)
        route.handler.ServeHTTP(w, req.WithContext(ctx))
        return
    }
    http.NotFound(w, req) // no route matched; serve a 404
}
```

This router is missing some important features: among other things, it doesn't do any kind of path normalization, it has no performance guarantees, and it doesn't properly handle URL and Regexp escaping and normalization. As such, unlike my usual advice, if you need more sophisticated routing than the stdlib can provide, **I suggest you use an external routing library rather than build it yourself. But just use a router** - don't use anything that forces you in to an entire ecosystem of libraries and frameworks.

The following program, `routerex`, implements a simple server that serves two endpoints. `GET /time` returns the current time in RFC3339 format, and `GET /panic` panics. Any other endpoint returns a 404.

```go
// TODO
```