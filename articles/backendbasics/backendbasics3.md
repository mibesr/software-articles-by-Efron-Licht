# backend basics, pt 3

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
// Log returns a RoundTripFunc that logs the request duration and status code.
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

// Log returns a middleware that injects a logger into the request context. It uses the trace from the context as a prefix, if it exists.
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