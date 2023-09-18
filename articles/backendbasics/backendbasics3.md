# Backend from the Beginning, part 3: Databases, Dependency Injection, Middleware, and Routing

A software article by Efron Licht.

September 2023

<<article list placeholder>>

This is the third article in a series on backend development in Go that aims to teach **understanding** of how backend is put together, rather than just lego-like assembly of pre-built components. This article will try to be accessible on it's own, but it will be much easier to follow if you've read the [first](https://eblog.fly.dev/backendbasics.html) and [second](https://eblog.fly.dev/backendbasics2.html) articles in the series.

A brief note before we begin: it's wild how well the first two articles went over - 60,000 views and counting from reddit alone! An encouraging sign, to say the least. Anyways, let's get to it.

### Review

- In the [first article](https://eblog.fly.dev/backendbasics.html), we went through the components of the internet up to HTTP - TCP, IP, & DNS, and built our own HTTP library and basic servers.

- In the [second article](https://eblog.fly.dev/backendbasics2.html), we went on a tour through the standard library's networking packages and demonstrated how to build HTTP servers and clients in practice, rather than theory.

- In this third article, we'll put the pieces together by covering our final missing pieces.
  - the `database/sql` package (how to connect to a database)
  - dependency injection for clients & servers (how to put a database in our server)
  - middleware (adding sophisticated behavior to our servers and clients, like authentication, logging, and tracing)
  - **routing** requests to the correct handler.

- The fourth article will be an opinion piece on software design and frameworks in the large. It may not come out; only if I think I can do it justice rather than make Yet Another Software Rant.

## Databases

Sooner or later your backend project will need to store data that falls into one or more of the following categories:

- **Persistent** - data that needs to be stored for longer than the lifetime of a single process; i.e, across program or system restarts.
- **Shared** - data that needs to be accessed by multiple processes or systems.
- **Large** - data that's too large to keep in memory.

There are a variety of ways to solve these problems, including but not limited to:

- writing to files (either locally, using a networked filesystem, or something like aws s3)
- using a key-value store like redis or memcached
- using a traditional relational database like postgres or mysql

But traditional relational databases are by far the most common, so we'll focus on that for now. Specifically, we will use [PostgreSQL](https://www.postgresql.org/), a popular open-source relational database. `SQL` is too large of a topic to cover in this article, so I'll assume you have some knowledge and stick to the parts about integrating it with Go and backend development specifically. I hope to eventually do an article series covering SQL and databases more in-depth.

## Connecting to a Database

Like most databases, **PostgreSQL** is a client-server application that uses the TCP/IP stack to communicate with clients.  We covered the TCP/IP stack in the [first article](https://eblog.fly.dev/backendbasics.html), so we won't go into detail here. Unlike our servers so far, which used `HTTP` on top of that, `postgres` uses it's own binary application-level protocol.If we wanted to communicate with postgres directly, the process would look broadly similar to the HTTP-by-hand we did in the [first article](https://eblog.fly.dev/backendbasics.html). Starting with a URL like "postgres://user:password@host:port/database", we'd do something like this:

- Use `net.ResolveTCPAddr` to resolve the hostname and port to an IP address.
- Use `net.DialTCP` to connect to the server.
- Spin up a goroutine to listen for responses from the server, and another to send requests to the server, communicating between them using channels or another synchronization primitive.

Unlike HTTP, however, the postgres protocol is not human-readable, and parsing the requests and responses would be a lot of work, so this time we'll skip straight to using libraries. Specifically, the `database/sql` package from the standard library.

### [`database/sql`](https://pkg.go.dev/database/sql)

The database/sql package provides a unified interface for interacting with SQL databases. Each different database needs it's own driver, which is registered at import time by convention. A couple of notes before we get started:

- Even though the overall interface is generic, your queries must be written in the SQL dialect of the database you're using; there's no abstraction layer for that. In particular, watch out for differences in parameter placeholders: postgres uses `$1`, `$2`, etc, while mysql uses `?`.
- Similarly, the error messages are specific to the database you're using and not particularly helpful. You'll need to read the documentation for your database to understand what they mean.

While Go provides a standard _interface_ for interacting with databases, it does not provide _drivers_ to make the actual connections and translate the requests and responses to and from the database's protocol. That's the domain of third-party drivers. The [`github.com/jackc/pgx/v5`](https://pkg.go.dev/github.com/jackc/pgx/v5) provides such a driver for postgres.

If we want to use it, we'll need to import that package, like so:

```go
import (
    "database/sql"
    _ "github.com/jackc/pgx/v5" // register the driver
)
```

Note the '`_`' : this is a "blank import", which means that we're importing the package for it's side effects (that is, registering the driver).

### sql.Open

The `sql.Open` function connects to a database and returns a `*sql.DB` object that we can use to interact with it. It takes two arguments: the name of the driver, and a connection string. The connection string is driver-specific, but for postgres it looks like this:

`"postgres://user:password@host:port/database?sslmode=mode"`

That is, it's a URL with the following required parameters:

- `user` - the username to connect with
- `password` - the password to connect with
- `host` - the hostname of the database server
- `port` - the port to connect to
- `database` - the name of the database to connect to

And a query parameter:

- `mode` - whether to use SSL to connect to the database. This is required for most hosted databases, but we'll disable it for now.

We covered URLs in detail in the [first article](https://eblog.fly.dev/backendbasics.html), so this should be pretty familiar.

By convention, `postgres` uses port `5432`. Usually, you'll want to store either the entire connection string or the individual components in environment variables, so that you

- can change them without recompiling your program
- don't leak secrets like passwords into your source code or compiled binaries.

The following table shows the environment variables we'll use, and their corresponding components of the connection string:
| env var | connection string component | note
| --- | --- | --- |
| `PG_USER` | `user` |
| `PG_PASSWORD` | `password` | be careful not to leak this!
| `PG_HOST` | `host` |
| `PG_PORT` | `port` | integer in range 0-65535; 5432 for postgres by convention
| `PG_DATABASE` | `database` |
| `PG_SSLMODE` | `mode` | `disable` or `require`; optional

### Sidenote: configuration

A brief note on configuration: as backend programs grow, they tend to accumulate a lot of configuration, which if not carefully managed can make programs fail in mysterious ways.

**It's always good to let the user know which configuration knobs they're missing, rather than just failing with a cryptic error message.** Configuration struggles are a common source of frustration for developers: spending a little bit of time early on error messages will save you a lot of time in the long run.

I've written a handy library for environment variables, [`enve`](https://pkg.go.dev/gitlab.com/efronlicht/enve) for this purpose, but for now, we'll do it by hand: the concept is easy.

Installing and configurating databases can be a bit tricky, so for the purpose of this article, we'll use the wonderful [`fergusstrange/embedded-postgres`](https://github.com/fergusstrange/embedded-postgres) to stick the database directly in our binary. This obviously isn't suitable for production, since you'll have no persistent storage, but it's great for testing and development, and means that my examples will work right out of the box for you on a variety of platforms.

The following complete program, `dbping`, sets up an embedded postgres database and connects to it, pinging it to make sure we can connect.

#### DB Example 1: `dbping`

```go
// dbping.go
package main

import (
    "context"
    "database/sql"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "sort"
    "strconv"
    "time"

    embeddedpostgres "github.com/fergusstrange/embedded-postgres" // embedded postgres server.
    _ "github.com/jackc/pgx/v5"                                   // register the db driver
)


func main() {
    timeout := flag.Duration("timeout", 5*time.Second, "timeout for connecting to postgres")
    flag.Parse()

    cfg, err := pgConfigFromEnv() // defined below
    if err != nil {
        log.Fatalf("postgres configuration error: %v", err)
    }
    // ---- setup embedded postgres server ----
    portN, err := strconv.Atoi(cfg.port)
    if err != nil {
        panic(err)
    }

    // we'll mirror the postgres config in the environment so that you can't actually get it 'wrong' when running
    // this example; you do need to set the environment variables, though.
    embeddedCfg := embeddedpostgres.DefaultConfig().
        Username(cfg.user).
        Password(cfg.password).
        Database(cfg.database).
        Port(uint32(portN)).
        Logger(io.Discard) // discard embedded postgres' logs; they're not helpful for this example

    embeddedDB := embeddedpostgres.NewDatabase(embeddedCfg)
    if err := embeddedDB.Start(); err != nil {
        panic(err)
    }
    log.Printf("postgres is running on: %s\n", embeddedCfg.GetConnectionURL())
    defer embeddedDB.Stop() // if we don't stop the database, it will continue running after our program exits and block the port.

    // ---- connect to postgres ----

    db, err := sql.Open(
        "postgres", 
        cfg.String(), // defined below
    )
    if err != nil {
        panic(err)
    }
    defer db.Close() // always close the database when you're done with it.

    // always ping the database to ensure a connection is made.
    // any time you talk to a DB, use a context with a timeout, since DB connections could be lost or delayed indefinitely.
    ctx, cancel := context.WithTimeout(context.Background(), *timeout)
    defer cancel()
    if err := db.PingContext(ctx); err != nil {
        panic(err)
    }
    log.Println("ping successful")

}

// pgconfig is a struct that holds the configuration for connecting to a postgres database.
// each field corresponds to a component of the connection string.
// the following required environment variables are used to populate the struct:
//
//    PG_USER
//     PG_PASSWORD
//     PG_HOST
//     PG_PORT
//     PG_DATABASE
//
// additionally, the following optional environment variable is used to populate the sslmode:
//
//    PG_SSLMODE: must be one of  "", "disable", "allow", "require", "verify-ca", or "verify-full"
type pgconfig struct {
    user, database, host, password, port string // required
    sslMode                              string // optional
}

func pgConfigFromEnv() (pgconfig, error) {
    var missing []string
    // small closures like this can help reduce code duplication and make intent clearer.
    // you generally pay a small performance penalty for this, but for configuration, it's not a big deal;
    // you can spare the nanoseconds.
    // i prefer little helper functions like this to a complicated configuration framework like viper, cobra, envconfig, etc.
    get := func(key string) string {
        val := os.Getenv(key)
        if val == "" {
            missing = append(missing, key)
        }
        return val
    }
    cfg := pgconfig{
        user:     get("PG_USER"),
        database: get("PG_DATABASE"),
        host:     get("PG_HOST"),
        password: get("PG_PASSWORD"),
        port:     get("PG_PORT"),
        sslMode:  os.Getenv("PG_SSLMODE"), // optional, so we don't add it to missing
    }
    switch cfg.sslMode {
    case "", "disable", "allow", "require", "verify-ca", "verify-full":
        // valid sslmode
    default:
        return cfg, fmt.Errorf(`invalid sslmode "%s": expected one of "", "disable", "allow", "require", "verify-ca", or "verify-full"`, cfg.sslMode)
    }

    if len(missing) > 0 {
        sort.Strings(missing) // sort for consistency in error message
        return cfg, fmt.Errorf("missing required environment variables: %v", missing)
    }
    return cfg, nil
}

// String returns the connection string for the given pgconfig.
func (pg pgconfig) String() string {
    s := fmt.Sprintf("postgres://%s:%s@%s:%s/%s", pg.user, pg.password, pg.host, pg.port, pg.database)
    if pg.sslMode != "" {
        s += "?sslmode=" + pg.sslMode
    }
    return s
}

```

Let's build and run it:

```bash
go build -o dbping ./dbping.go
./dbping
```

OUT:

```text
2023/09/17 09:52:45 missing 5 required environment variable(s): [PG_DATABASE PG_HOST PG_PASSWORD PG_PORT PG_USER]
```

Whoops; we forgot to set the environment variables. Good thing we added those error messages.
Let's try again:

```bash
PG_USER=postgres PG_PASSWORD=admin PG_HOST=localhost PG_PORT=5432 PG_DATABASE=postgres ./dbping
```

OUT:

```text
panic: pq: SSL is not enabled on the server
```

SSL, or Secure Sockets Layer, is a protocol for encrypting network traffic; it's the **S** in HTTPS. (This article series won't get into SSL and HTTPS, but you should, on your own time.) Let's set the PG_SSLMODE environment variable to "disabled":

```bash
PG_USER=postgres PG_PASSWORD=admin PG_HOST=localhost PG_PORT=5432 PG_DATABASE=postgres PG_SSLMODE=disable ./dbping
```

OUT:

```text
2023/09/17 10:14:14 postgres configuration error: invalid sslmode "disabled": expected one of "", "disable", "allow", "require", "verify-ca", or "verify-full"
```

... looks like it's _disable_, not _disable**d**_. One last time:

```bash
PG_USER=postgres PG_PASSWORD=admin PG_HOST=localhost PG_PORT=5432 PG_DATABASE=postgres PG_SSLMODE=disable ./dbping
```

OUT:

```text
2023/09/17 10:15:19 postgres is running on: postgresql://postgres:admin@localhost:5432/postgres
2023/09/17 10:15:19 ping successful
```

OK, looks good. Little configuration errors like this can easily stall a project for hours or days, so it's worth taking the time to make sure your error messages are clear and helpful. **If you run into a configuration error, take the time to add a message that guides your next user to the solution; after all, that next user might be you.**

### Using `*sql.DB`

The following table summarizes the basic API of `*sql.DB`. Note that all methods take a `context.Context` as their first argument. **Never use the non-`Context` versions of these methods; they are a deprecated API. If you're not sure what context to use, use `context.TODO()`**.

| Method | Returns | Description | Use Cases |
| --- | --- | --- |  --- |
| `PingContext`| error |  Ping the database to ensure a connection is made. | Health check
| `ExecContext` | Result, error | Execute a query that does not return rows. | Create, Update, Delete
| `QueryRowContext` | Row | Execute a query that returns a single row.| Single item lookup
| `QueryContext` | Rows, error | Execute a query that returns rows. | All other queries

I had a big section here where I demonstrated the APIs, but it quickly grew so big it completely overwhelmed the rest of the article, which is already over twice as long as part 2. Instead, I'll just point you to the [official docs for the `database/sql` package](https://pkg.go.dev/database/sql)

### Dependency Injection, or, "how do I put a database in my server?"

So far when we've created http handlers, they've been self-contained: they don't depend on anything outside of themselves; that is, they're just `func(http.ResponseWriter, *http.Request)`s. But in the real world, we'll want to access a database, cache, message queue, or other outside dependency from within our handlers.

The simplest and best way to handle this is to **pass the dependencies in as arguments** to a function that creates the handler.

That is, instead of:

```go
// example: 'global' database connection
var db *sql.DB 
func init() {
    db, err := sql.Open("postgres", "...")
    if err != nil {
        panic(err)
    }
}
func getUser(w http.ResponseWriter, r *http.Request) {
    // ... parse & validate request...


    if err :=db.QueryRowContext(r.Context(), "SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Email); err != nil {

    }
    // etc, etc, etc
}
```

_inject_ the dependency:

```go
// this function RETURNS a handler, rather than being a handler itself.
func getUser(db *sql.DB) http.HandlerFunc {
    // db is now a local variable, rather than a global variable.

    // this is the actual handler function, sometimes called a 'closure' since it "closes over" the db variable.
    return func(w http.ResponseWriter, r *http.Request) {
        // ... parse & validate request...
        if err :=db.QueryRowContext(r.Context(), "SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Email); err != nil {
            // ...
        }
    }
}
```

Alternatively, you can declare a struct containing the dependencies and let that struct implement the `http.Handler` interface:

```go
type userHandler struct { db *sql.DB }
func (u userHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // ... parse & validate request...
    if err :=u.db.QueryRowContext(r.Context(), "SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name, &user.Email); err != nil {
        // ...
    }
}
```

I recommend you stick with closures, since they are lighter-weight: the code is only in one place rather than two, and you don't need to create a new struct type for each handler.

We'll use dependency injection repeatedly throughout the rest of this article, especially while writing middleware.

## middleware

### client middleware

There's a lot of shared behavior we might want to add to many of our outgoing HTTP requests.

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

What we really need is some kind of composability, where we can quickly apply _some_ of the options to a client on an as-needed basis. We can build such a system by using _middleware_. Let's talk about how `http.Client` works first:
when we call `Client.Do`, the client sends a request to the server by calling the `RoundTrip` method on it's `http.RoundTripper`, which is usually `http.DefaultTransport`. That `RoundTrip` method does all the low-level work of sending the request and receiving the response that we covered in the [first article](https://eblog.fly.dev/backendbasics.html) (though, admittedly, in a much more sophisticated way).

If we substituted out that `RoundTripper` for our own, we could intercept the request and modify it before it's sent to the server. We could also intercept the response and modify it before it's returned to the caller. We'd just have to make sure to eventually call the original `RoundTrip` method, so that the request actually gets sent to the server.

That's exactly what **middleware** does. Essentially, middleware "wraps" a client, sitting between it and the outside world. It modifies requests and responses as they pass through it, and can short-circuit the request/response cycle entirely.

Our desired API will look something like this:

```go
var rt http.RoundTripper = http.DefaultTransport
rt = TimeRequest(rt) 
rt = RetryOn5xx(rt, 10*time.Millisecond, 3)
rt = ...
client := &http.Client{
    Timeout: 1 * time.Second,
    Transport: rt,
}
```

### Building Client Middleware

The RoundTripper interface looks like this:

```go
type RoundTripper interface {
    RoundTrip(*http.Request) (*http.Response, error)
}
```

We'll follow the example of the `http.HandlerFunc` and build a `RoundTripFunc` type that implements this interface:

```go

// RoundTripFunc is an adapter to allow the use of ordinary functions as RoundTrippers, a-la http.HandlerFunc
type RoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface by calling f(r)
func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {return f(r)}

var _ http.RoundTripper = RoundTripFunc(nil) // assert that RoundTripFunc implements http.RoundTripper at compile time
```

#### Client Middleware without Dependencies

Let's build a package, `clientmw`, and implement a few simple middlewares. Each middleware will be a function that takes a `http.RoundTripper` and returns a `RoundTripFunc` that wraps it.

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

#### Client Middleware with Injected Dependencies

But generally speaking you'll want to pass some arguments to your middleware, _injecting_ the dependencies.

For example, we might want to have configurable retries:

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
package trace
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

Let's put together a `Trace` middleware that adds a `Trace` struct to the context and adds the `X-Trace-ID` and `X-Request-ID` headers to the request.

```go
package clientmw
// Trace returns a RoundTripFunc that 
// - adds a trace to the request context
// - generating a new one if necessary
// - adds the X-Trace-ID and X-Request-ID headers to the request
// - then calls the next RoundTripper
func Trace(rt http.RoundTripper) RoundTripFunc {
    return func(r *http.Request) (*http.Response, error) {
        defer logExec("Trace")()
        // does the request already have a trace? if so, use it. otherwise, generate a new one.
        traceID, err := uuid.Parse(r.Header.Get("X-Trace-ID"))
        if err != nil {
            traceID = uuid.New()
        }

        // build the trace. it's a small struct, so we put it directly in the context and don't bother with a pointer.
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

Let's pick up this trace in the next middleware, one that adds a logger to our requests. We'll just use the standard library's unstructured [`log`](https://pkg.go.dev/log) package for now.

> Note: In practice you should probably use a structured logger. Both [`rs/zerolog`](https://github.com/rs/zerolog) and [`uber-go/zap`](https://github.com/uber-go/zap) are popular choices, and the standard library has recently introduced it's own structured logging package, [`log/slog`](https://pkg.go.dev/log/slog). I can happily recommend any of these. But for now, we'll dodge the question entirely and leave logging and metrics for a future article.

This will supersede our original `TimeRequest` middleware, so we'll add the timing logic here as well.

```go
package clientmw
// Log returns a RoundTripFunc that logs the request duration and status code. It uses the trace from the context as a prefix, if it exists. See Trace in this package and servermw.Log for the server-side implementation.
func Log(rt http.RoundTripper, log *log.Logger) RoundTripFunc {
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

> A brief final note on client middleware: the [documentation for RoundTripper](https://pkg.go.dev/net/http#RoundTripper) says that it shouldn't modify the request or response. I disagree with this; it's simpler and easier to intercept the RoundTripper than to build another layer _on top_ of **http.Client**. Over the years, this seems to be the consensus for backend development in Go. If you disagree, you can always build your own layer **on top** of http.Client that wraps it's `Do()` method instead, like so:
>
>   ```go
>   type HTTPDoer interface { Do(*http.Request) (*http.Response, error) }
>   type HTTPDoerFunc func(*http.Request) (*http.Response, error)
>   func (f HTTPDoerFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }
>   var _ HTTPDoer = HTTPDoerFunc(nil) // assert that HTTPDoerFunc implements HTTPDoer at compile time
>   var _ HTTPDoer = (*http.Client)(nil) // assert that http.Client implements HTTPDoer at compile time
>   ```
>

### Server Middleware

Server middleware is very similar to Client middleware. Rather than wrapping a `RoundTripper`, we wrap a `http.Handler`. [We covered this in the last article](https://eblog.fly.dev/backendbasics2.html), but let's briefly review:

The [`http.Handler`](https://pkg.go.dev/net/http#Handler) interface looks like this:

```go
type Handler interface {
    ServeHTTP(http.ResponseWriter, *http.Request)
}
```

We don't need to define our own `HandlerFunc` type, because [the standard library already provides one](https://pkg.go.dev/net/http#HandlerFunc).

```go
// HandlerFunc adapts a function to work as a http.Handler.
type HandlerFunc func(http.ResponseWriter, *http.Request)
// ServeHTTP calls f(w, r)
func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) { f(w, r) }
```

We can use this to build our own middleware by wrapping handlers in a closure and returning a `HandlerFunc`, the same way wrapped `RoundTripper`s in a closure and returned a `RoundTripFunc`.

Let's add traces and logs to our server. The implementation is broadly symmetrical to the client middleware:

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

This `RecordingResponseWriter` is broadly similar to the one implemented by the standard library's [`httptest.ResponseRecorder`](https://pkg.go.dev/net/http/httptest#ResponseRecorder). As usual, Go's standard library uses a small set of simple interfaces to cover a wide range of use cases.

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

Looks like everything is working as expected. So far, we've covered (nearly) everything you might have used a framework for:

- Requests
- Responses
- Middleware
- Serialization
- Dependency Injection

But we haven't covered **routing** yet. Let's fix that.

### Routing

**Routing** is the process of matching a request to a handler via it's `METHOD` and `PATH`. In Go, there's nothing particularly special about routing: it's just something that the `Handler` inside your `Server` does.

The most basic kind of routing is just a `switch` statement, like we saw above. That only dealt with paths, but routing based off `METHOD` is just as easy: the following code is the 'router' that serves **the website you're reading this on**.

```go
var router http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
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
                static.ServeFile(w, r) // serve pre-zipped, embedded files
                return
            }
}
```

**This is all you need for small programs.** For convenience, Go's stdlib comes with a built-in Router, [`http.ServeMux`](https://pkg.go.dev/net/http#ServeMux), which uses a simple prefix-based matching scheme: the longest prefix that matches the request path wins. It's implemented like this:

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

ServeMux is perfectly fine for the vast majority of backend programs, but is not very flexible: it doesn't even support routing by METHOD. (There is an [accepted proposal](https://github.com/golang/go/issues/61410) to add both this and wildcards to `http.ServeMux`, but it's not implemented yet.)

More complicated routers, like the popular [gorilla/mux](https://github.com/gorilla/mux), allow for routing by method, by patterns matching regular expressions, and for extracting variables from the URL path. I'll quote their documentation here to give you an idea of what this looks like:

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

[See the full code, including 100% test coverage, at https://go.dev/play/p/BBGLxqepogO](https://go.dev/play/p/BBGLxqepogO).

The following table summarizes our external API:

| Type/Func | Description |
| --- | --- |
| `Router` | A router that matches HTTP requests to handlers based on the request path. |
| `Router.AddRoute(pattern, method, handler)` | Add a route to the router. |
| `Router.ServeHTTP(w, r)` | Serves the request by matching it against the routes in the router. |
| `PathVars` | A `map[string]string` of path params to their values |
| `Vars(r)` | Returns the path parameters for the current request, or an empty map if there are none. |

Each route will contain a `method` (e.g, `GET`), a `pattern` (e.g, `/articles/{category}/{id:[0-9]+}`), and a `http.Handler`. Patterns that match a regular expression corresponding to the `pattern` will be dispatched to the `handler`.

That is, we'll need to convert a pattern like `/articles/{category}/{id:[0-9]+}` into a regular expression like `^/articles/([a-zA-Z]+)/([0-9]+)$`, and extract the path parameters from the request path, so that the _server_ can find what we meant by "category" and "id".

Each route will look like this:

```go
type route struct {
    pattern *regexp.Regexp // the compiled regexp
    names   []string // the names of the path parameters; one per capture group in the regexp
    raw     string // the raw pattern string
    method  string // the HTTP method to match; if empty, all methods match.
    handler http.Handler // underlying handler
}

```

We'll define a `Router` type that contains a slice of routes, matching against each in turn:

```go
// Router allows you to match HTTP requests to handlers based on the request path.
// It use a syntax similar to gorilla/mux:
// /path/{regexp}/{name:captured-regexp}
// AddRoute adds a route to the router.
// Vars returns the path parameters for the current request, or nil if there are none.
type Router struct {routes []route}
```

We'll need to compile the pattern into a regular expression, and extract the path parameters from the pattern.

This entails keeping track of which names correspond to which capture groups in the regexp.

While we could use named capture groups (`(?P<name>...)`), this is slow and bulky and requires extra iteration. Instead, we'll just keep track of the names in a slice, and use the slice index to determine which capture group corresponds to which name.

That is,suppose we have pattern

```regexp
/chess/replay/{white:[a-zA-Z]+}/{black:[a-zA-Z]+}/{id:[0-9]+}
```

This should compile into the regexp

```regexp
^/chess/replay/([a-zA-Z]+)/([a-zA-Z]+)/([0-9]+)$
```

With the names slice

```go
[]string{"white", "black", "id"}
```

We define a function, `buildRoute`, to do this:

```go
func buildRoute(pattern string) (re *regexp.Regexp, names []string, err error) {
    if pattern == "" || pattern[0] != '/' {
        return nil, nil, fmt.Errorf("invalid pattern %s: must begin with '/'", pattern)
    }
    var buf strings.Builder
    buf.WriteByte('^') // match the beginning of the string

    // we gradually build up the regexp, and keep track of the path parameters we encounter.
    // e.g, on successive iterations, we'll have:
    // FOR {
    // 0: /chess, nil
    // 1: /chess/replay, nil
    // 2: /chess/replay/([a-zA-Z]+), [white]
    // 3: /chess/replay/([a-zA-Z]+)/([a-zA-Z]+), [white, black]
    // 4: /chess/replay/([a-zA-Z]+)/([a-zA-Z]+)/([0-9]+), [white, black, id]
    // }
    for _, f := range strings.Split(pattern, "/")[1:] {
        buf.WriteByte('/')                                    // add the '/' back
        if len(f) >= 2 && f[0] == '{' && f[len(f)-1] == '}' { // path parameter
            trimmed := f[1 : len(f)-1] // strip off the '{' and '}'
            // - {white:[a-zA-Z]+} -> [a-zA-Z]+
            if before, after, ok := strings.Cut(trimmed, ":"); ok { // its a regexp-capture group
                names = append(names, before)

                // replace with a capture group: i.e, if we have {id:[0-9]+}, we want to replace it with ([0-9]+)
                buf.WriteByte('(') //
                buf.WriteString(after)
                buf.WriteByte(')')
                // white:[a-zA-Z]+ -> ([a-zA-Z]+)
            } else {
                buf.WriteString(trimmed) // a regular expression, but not a captured one
            }
        } else {
            buf.WriteString(regexp.QuoteMeta(f)) // escape any special characters
        }

    }
    // check for duplicate path parameters
    for i := range names {
        for j := i + 1; j < len(names); j++ {
            if names[i] == names[j] {
                return nil, nil, fmt.Errorf("duplicate path parameter %s in %q", names[i], pattern)
            }
        }
    }
    buf.WriteByte('$') // match the end of the string
    re, err = regexp.Compile(buf.String())
    if err != nil {
        return nil, nil, fmt.Errorf("invalid regexp %s: %w", buf.String(), err)
    }
    return re, names, nil
}
```

We'll obtain path parameters on the server by creating a map of path parameters to their values, and adding it to the request context, reusing the `ctxutil` package we wrote earlier.

Then, we'll store those path parameters in a `map[string]string` and put the map in the context. We'll use a unique type for the map so that we can use `ctxutil.Value` to retrieve it without stepping on any other context values. (A collision is unlikely, but it's good practice to avoid it anyway; types are free.)

```go
// Vars is a map of path parameters to their values. It is a unique type so that ctxutil.Value can be used to retrieve it.
type PathVars map[string]string

var empty = make(PathVars)
// Vars returns the path parameters for the current request. It will be nil if you didn't use a router to serve the request.
func Vars(ctx context.Context) PathVars { v, _ := ctxutil.Value[PathVars](ctx); return v }
// AddRoute adds a route to the router. Method is the HTTP method to match; if empty, all methods match.
// Method will be converted to uppercase and trimmed of whitespace, so
// "get", "gEt", " geT", and "GET" are all equivalent.
```

Adding routes is straightforward:

```go
// AddRoute adds a route to the router. Method is the HTTP method to match; if empty, all methods match.
func (r *Router) AddRoute(pattern string, h http.Handler, method string) error {
    re, names, err := buildRoute(pattern)
    if err != nil {
        return err
    }
    r.routes = append(r.routes, route{
        raw:     pattern,
        pattern: re,
        names:   names,
        method:  strings.ToUpper(strings.TrimSpace(method)),
        handler: h,
    })

    // sort the routes by length, so that the longest routes are matched first.
    sort.Slice(r.routes, func(i, j int) bool {
        return len(r.routes[i].raw) > len(r.routes[j].raw) || (len(r.routes[i].raw) == len(r.routes[j].raw) && r.routes[i].raw < r.routes[j].raw) // sort by length, then lexicographically
    })
    return nil
}
```

Building the routes was the hard part; actual dispatch is easy. We pick the first route that matches the request path, add the relevant path segments to a map in the context, and serve the request using the associated handler.

```go

// pathVars extracts the path parameters from the path and into a map. leave it as-is.
func pathVars(re *regexp.Regexp, names []string, path string) PathVars {
    matches := re.FindStringSubmatch(path)
    if len(matches) != len(names)+1 { // +1 because the first match is the entire string
        panic(fmt.Errorf("programmer error: expected regexp %q to match %q", path, re.String()))
    }
    vars := make(PathVars, len(names))
    for i, match := range matches[1:] { // again, skip the first match, which is the entire string
        vars[names[i]] = match
    }
    return vars
}

// ServeHTTP implements http.Handler, dispatching requests to the appropriate handler.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    for _, route := range rt.routes {
        if route.pattern.MatchString(r.URL.Path) && (route.method == "" || route.method == r.Method) {
            vars := pathVars(route.pattern, route.names, r.URL.Path)
            ctx := CtxWithVal(r.Context(), vars)
            route.handler.ServeHTTP(w, r.WithContext(ctx))
            return
        }
    }
    http.NotFound(w, r) // no route matched; serve a 404
}

```

This router is missing some important features: among other things, it doesn't do any kind of path normalization, it has no performance guarantees, and it doesn't properly handle URL and Regexp escaping and normalization. As such, unlike my usual advice, if you need more sophisticated routing than the stdlib can provide, **I suggest you use an external routing library rather than build it yourself. But just use a router** - don't use anything that forces you in to an entire ecosystem of libraries and frameworks.

That being said, this router is perfectly fine for small programs, and more importantly, _now you know how they work_, so you can figure out how to fix them if they go wrong.

### Graduation: Putting it all together

Let's write ourselves a client and server that puts together all the pieces we've written so far. Our server will route to multiple endpoints. Some of our routes will take a POST json body, some will take query parameters, and still others will take path parameters. We'll use all the middleware we've written so far in order to log, trace, and recover from panics.

The following program, `graduation`, implements a complete web server that demonstrates all of the concepts we've covered so far.

First, we'll use our router to register all the endpoints we want to serve.

```go
package main
// register routes.
func buildBaseRouter() (*Router, error) {
    var r = new(Router) // we'll add routes to this router.
    /* --- design note: ---
    you could just add the routes on a separate line for each, 
    but I like building the slice of routes and iterating over it; 
    it makes the essential similarity of each route more obvious.
    */
    for _, route := range []struct {
        pattern, method string
        handler         http.HandlerFunc
    }{
        // GET / returns "Hello, world!"

        {
            pattern: "/",
            method:  "GET",
            /* ----- design note: ----
                this route demonstrates the simplest possible handler.
                note the _ for the request parameter; we don't need it, so we don't bind it.
            -----------------------------*/
            handler: func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("Hello, world!\r\n")) },
        },

        // GET /panic always panics.

        {
            pattern: "/panic",
            method:  "GET",
            /* ----- design note: ----
                this route demonstrates how middleware can handle error conditions:
                the panic will be caught by Recovery, which will write a 500 status code and "internal server error" message to the response.
                rather than leaving the connection hanging.
                note the _ and _ for the request and response parameters; we don't need them, so we don't bind them.
            -----------------------------*/
            handler: func(_ http.ResponseWriter, _ *http.Request) { panic("oh my god JC, a bomb!") },
        },

        // POST /greet/json returns a JSON object with a greeting and a category based on the age.
        // it must be called with a JSON object in the form {"first": "efron", "last": "licht", "age": 32}
        {
            "/greet/json",
            "POST",
            /* ----- design note: ----
            // this route is a sophisticated example that has both path parameters (using our custom router) and query parameters.
            // it 'puts everything together' and demonstrates how to use the router and middleware together.
            // ---- */
            func(w http.ResponseWriter, r *http.Request) {

                var req struct {
                    First, Last string
                    Age         int
                }
                if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
                    WriteError(w, err, http.StatusBadRequest) // remember to return after writing an error!
                    return
                }
                if req.Age < 0 {
                    WriteError(w, errors.New("age must be >= 0"), http.StatusBadRequest)
                    return
                }
                var category string
                switch {
                case req.Age < 13:
                    WriteError(w, errors.New("forbidden: come back when you're older"), http.StatusForbidden)
                    return
                case req.Age < 21:
                    category = "teenager"
                case req.Age > 65:
                    category = "senior"
                default:
                    category = "adult"
                }
                _ = WriteJSON(w, struct {
                    Greeting string `json:"greeting"`
                    Category string `json:"category"`
                }{
                    fmt.Sprintf("Hello, %s %s!", req.First, req.Last),
                    category,
                })
            }},

        // GET /time returns the current time in the given format.
        // it demonstrates how to use query parameters.
        {
            pattern: "/time",
            method:  "GET",
            handler: func(w http.ResponseWriter, r *http.Request) {
                format := r.URL.Query().Get("format")
                if format == "" {
                    format = time.RFC3339
                }
                tz := r.URL.Query().Get("tz")
                var loc *time.Location = time.Local
                if tz != "" {
                    var err error
                    loc, err = time.LoadLocation(tz)
                    if err != nil {
                        WriteError(w, fmt.Errorf("invalid timezone %q: %w", tz, err), http.StatusBadRequest)
                        return
                    }
                }
                _ = WriteJSON(w, struct {
                    Time string `json:"time"`
                }{time.Now().In(loc).Format(format)})
            },
        },
        // GET /echo/{a}/{b}/{c} returns the path parameters as a JSON object in the form {"a": "value of a", "b": "value of b", "c": "value of c"}
        // the query parameter "case" can be "upper" or "lower" to convert the values to uppercase or lowercase.
        {
            pattern: "/echo/{a:.+}/{b:.+}/{c:.+}",
            method:  "GET",
            /* ----- design note: ----
            this route is a sophisticated example that has both path parameters (using our custom router)
            and query parameters.
            it 'puts everything together' and demonstrates how to use the router and middleware together.
            ---- */
            handler: func(w http.ResponseWriter, r *http.Request) {
                vars, _ := ctxutil.Value[PathVars](r.Context())
                switch strings.ToLower(r.URL.Query().Get("case")) {
                case "upper":
                    for k, v := range vars {
                        vars[k] = strings.ToUpper(v)
                    }
                case "lower":
                    for k, v := range vars {
                        vars[k] = strings.ToLower(v)
                    }
                }
                _ = WriteJSON(w, vars)
            },
        },
    } {
        if err := r.AddRoute(route.pattern, route.handler, route.method); err != nil {
            return nil, fmt.Errorf("AddRoute(%q, %v, %q) returned error: %v", route.pattern, route.handler, route.method, err)

        }
        log.Printf("registered route: %s %s", route.method, route.pattern)
    }
    return r, nil
}
```

In main, we'll take the router we just built and add some of the server middleware we previously wrote:

```go

func main() {
    port := flag.Int("port", 8080, "port to listen on")
    flag.Parse()

    h, err := buildBaseRouter()
    if err != nil {
        log.Fatal(err)
    }
    h = applyMiddleware(h)
```

Then we'll start up the server.

```go
    // build and start the server.
    // remember, you should always apply at least the Read and Write timeouts to your server.
    server := http.Server{
        Addr:         fmt.Sprintf(":%d", *port),
        Handler:      h,
        ReadTimeout:  1 * time.Second,
        WriteTimeout: 1 * time.Second,
    }
    log.Printf("listening on %s", server.Addr)
    go server.ListenAndServe()
    time.Sleep(20 * time.Millisecond)
}
```

The following program, `graduationdemo`, hits the server we just wrote with a variety of requests, demonstrating all the endpoints we just wrote.

```go
package main
func main() {
    var port int
    flag.IntVar(&port, "port", 8080, "port for outgoing requests")
    flag.Parse()
    rt = clientmw.Trace(rt)
    rt = clientmw.Log(rt)
    client := &http.Client{
        Transport: rt,
        Timeout:   1 * time.Second,
    }
    req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/panic", port), nil)
    if err != nil {
        log.Fatal(err)
    }

    resp, err := client.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("GET /panic:")
    resp.Write(os.Stdout)

    var paths = []string{
        "/",
        "/echo/first/second/third",
        "/echo/first/second/third",
        "/echo/first/second/third",
        "/time",
        "/time",
        "/time",
        "/time",
    }

    var queries = []map[string]string{
        nil,
        nil,
        {"case": "upper"},
        {"case": "lower"},
        nil,
        {"format": time.RFC1123},
        {"format": time.RFC1123, "tz": "America/New_York"},
        {"format": time.RFC1123, "tz": "America/Los_Angeles"},
        {"format": time.RFC1123, "tz": "America/Chicago"},
    }
    for i := range paths {
        q := make(url.Values)
        for k, v := range queries[i] {
            q.Set(k, v)
        }
        url := fmt.Sprintf("http://localhost:%d%s", port, paths[i])
        if len(q) > 0 {
            url += "?" + q.Encode()
        }
        ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            log.Fatal(err)
        }
        resp, err := client.Do(req)
        cancel()
        if err != nil {
            log.Fatal(err)
        }
        fmt.Printf("GET %s: \n", url)
        resp.Write(os.Stdout)
        fmt.Println("\n-------")
    }

}
```

If you run both programs, you should see something like this (lightly edited and heavily compressed for space):

```
server: GET /echo/first/second/third: [d4df5e13-fadf-4f4a-8eb9-7ca1dca0428b c16dc184-90ae-4a23-85cd-4898456ac25e]: 2023/09/18 10:59:04 200 OK: 39 bytes in 28.093µs
client: GET http://localhost:8080/echo/first/second/third: 2023/09/18 10:59:04 clientmw.go:112: 200 OK in 348.138µs
GET http://localhost:8080/echo/first/second/third: 
HTTP/1.1 200 OK
Content-Length: 39
Content-Type: application/json
Date: Mon, 18 Sep 2023 17:59:04 GMT

{"a":"first","b":"second","c":"third"}

server: GET /echo/first/second/third?case=upper: [2253f79f-e49d-455e-b32a-3d842c7f1ea5 bb748b7a-2a07-4d86-8992-0ba3f2e7423c]: 2023/09/18 10:59:04 200 OK: 39 bytes in 21.069µs
client: GET http://localhost:8080/echo/first/second/third?case=upper: 2023/09/18 10:59:04 clientmw.go:112: 200 OK in 342.51µs
GET http://localhost:8080/echo/first/second/third?case=upper: 
HTTP/1.1 200 OK
Content-Length: 39
Content-Type: application/json
Date: Mon, 18 Sep 2023 17:59:04 GMT

{"a":"FIRST","b":"SECOND","c":"THIRD"}

server: GET /time?format=Mon%2C+02+Jan+2006+15%3A04%3A05+MST&tz=America%2FLos_Angeles: [de41ddc1-fd01-4aa7-b666-a9a5d81a1c08 e7605ac9-aa5d-43b1-9fcc-5a90af39d737]: 2023/09/18 10:59:04 200 OK: 41 bytes in 21.901µs
client: GET http://localhost:8080/time?format=Mon%2C+02+Jan+2006+15%3A04%3A05+MST&tz=America%2FLos_Angeles: 2023/09/18 10:59:04 clientmw.go:112: 200 OK in 244.977µs
GET http://localhost:8080/time?format=Mon%2C+02+Jan+2006+15%3A04%3A05+MST&tz=America%2FLos_Angeles: 
HTTP/1.1 200 OK
Content-Length: 41
Content-Type: application/json
Date: Mon, 18 Sep 2023 17:59:04 GMT

{"time":"Mon, 18 Sep 2023 10:59:04 PDT"}
```

We'd like to have a better guarantee that our server is working correctly than just "it seems to work when I run it", though.

The [`httptest.Server`](https://pkg.go.dev/net/http/httptest#Server) listens on a random port and serves http using a provided handler.

The following heavily annotated test suite demonstrates how to use `http.Server` and table-driven tests to test our server.

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "net/http/httptest"
    "net/url"
    "os"
    "reflect"
    "strings"
    "testing"

    "gitlab.com/efronlicht/blog/articles/backendbasics/cmd/clientmw"
    "gitlab.com/efronlicht/blog/articles/backendbasics/cmd/ctxutil"
)

// initialized during TestMain.
var client *http.Client
var server *httptest.Server

// The TestMain function is a special function that runs before any tests are run; think of it as init()
// that only runs when you run tests.
func TestMain(m *testing.M) {
    router, err := buildBaseRouter()
    if err != nil {
        log.Fatal(err)
    }
    router = applyMiddleware(router) // apply middleware
    // httptest.NewServer starts an http server that listens on a random port.
    // you can use the URL field of the returned httptest.Server to make requests to the server.
    server = httptest.NewServer(router) // create a test server with the router

    // httptest.Server.Client() returns an http.Client that uses the test server and always accepts it's auth certificate.
    client = server.Client()

    if client.Transport == nil {
        client.Transport = http.DefaultTransport // use the default transport if the client doesn't have one.
    }
    // apply our client middleware to the client.
    client.Transport = clientmw.Log(clientmw.Trace(client.Transport)) // add logging and tracing to the client

    code := m.Run() // run the tests

    server.Close() // close the test server

    os.Exit(code) // exit with the same code as the tests; 0 if all tests passed, non-zero otherwise.
}

// TestNotFound tests that the router returns a 404 status code for requests that don't match any routes.
func TestNotFound(t *testing.T) {
    for _, tt := range []struct {
        method, path string
    }{
        {"DELETE", "/"},
        {"GET", "/notfound"},
        {"GET", "/chess/replay/efronlicht/bobross/1234"},
    } {
        req, _ := http.NewRequest(tt.method, server.URL+tt.path, nil)

        if resp, err := client.Do(req); err != nil {
            t.Errorf("client.Do(%q, %q) returned error: %v", tt.method, tt.path, err)
        } else if resp.StatusCode != http.StatusNotFound {
            t.Errorf("client.Do(%q, %q) returned status %d, want %d", tt.method, tt.path, resp.StatusCode, http.StatusNotFound)
        }

    }
}

// TestGraduation tests that the server works as expected.
// This is meant to demonstrate how to write tests for a server in a way that doesn't have too many dependencies
// or use any external libraries.
func TestGraduation(t *testing.T) {

    defer server.Close()

    // table-based testing is a common pattern in Go.
    for _, tt := range []struct {
        method, path string            // where is the request going?
        body         map[string]any    // what is the request body, if any? nil if no body.
        queries      map[string]string // what are the query parameters, if any? nil if no query parameters.
        wantStatus   int               // what status code do we expect?
        // what substrings do we expect to find in the response body?
        /* ----- design note: ----
        we're not testing the entire response body, because it's too brittle; small changes in the response body
        like whitespace or punctuation will cause the test to fail.
        instead, we test for substrings that we expect to find in the response body.
        this is a good compromise between testing the entire response body and testing nothing.
        if you have structured content, like JSON, you can unmarshal the response body into a struct and test that.
        */
        wantBodyContains []string
    }{
        { // GET / returns "Hello, world!"
            method:           "GET",
            path:             "/",
            wantStatus:       http.StatusOK,
            wantBodyContains: []string{"Hello, world!"},
        },
        { // GET /panic always panics.
            method:           "GET",
            path:             "/panic",
            wantStatus:       http.StatusInternalServerError,
            wantBodyContains: []string{"Internal Server Error"},
        },
        // POST /greet/json returns a JSON object with a greeting and a category based on the age,
        // and it's forbidden to children under 13.
        {
            method:           "POST",
            path:             "/greet/json",
            body:             map[string]any{"first": "Raphael", "last": "Frasca", "age": 10}, // my nephew turned 10 today! ;P
            wantStatus:       http.StatusForbidden,
            wantBodyContains: []string{"forbidden"},
        },
        // adults should get a greeting based on their age: i'm an adult.
        {
            method:           "POST",
            path:             "/greet/json",
            body:             map[string]any{"first": "Efron", "last": "Licht", "age": 32},
            wantStatus:       http.StatusOK,
            wantBodyContains: []string{"Efron", "Licht", "adult"},
        },
        // GET /time returns the current time in UTC, or in the timezone specified by the tz query parameter.
        {
            method:     "GET",
            path:       "/time",
            queries:    map[string]string{"tz": "America/New_York"},
            wantStatus: http.StatusOK,
            /* wantBodyContains: []string{"-4"}
            this test is bad, because it assumes that the test is running in a timezone that is 4 hours behind UTC;
            America/New_York is only 4 hours behind UTC during daylight savings time; otherwise it's 5 hours behind.
            be careful when writing tests that depend on timezones! */
        },
        // GET /time returns a 400 status code if the tz query parameter is invalid.
        {
            method:     "GET",
            path:       "/time",
            queries:    map[string]string{"tz": "invalid"},
            wantStatus: http.StatusBadRequest,
        },
    } {
        tt := tt // capture the range variable
        if len(tt.queries) != 0 {
            query := make(url.Values, len(tt.queries))
            for k, v := range tt.queries {
                query.Set(k, v)
            }
            tt.path += "?" + query.Encode()
        }
        // give the test a name that describes the request so we can easily see what passed and failed in the output:
        // i.e, TestGraduation/GET/ -> 200-OK
        // TestGraduation/POST/greet/json->403-Forbidden
        name := fmt.Sprintf("%s%s->%d-%s", tt.method, tt.path, tt.wantStatus, strings.ReplaceAll(
            http.StatusText(tt.wantStatus), " ", "-"))
        t.Run(name, func(t *testing.T) {
            path := server.URL + tt.path

            var body io.Reader
            if tt.body != nil {
                b, _ := json.Marshal(tt.body)
                body = bytes.NewReader(b)
            }

            req, err := http.NewRequestWithContext(context.TODO(), tt.method, path, body)
            if err != nil {
                t.Errorf("http.NewRequestWithContext(%q, %q, %v) returned error: %v", tt.method, tt.path, tt.body, err)
            }

            resp, err := client.Do(req)
            if err != nil {
                t.Errorf("client.Do(%q, %q) returned error: %v", tt.method, tt.path, err)
            }

            if resp.StatusCode != tt.wantStatus {
                t.Errorf("router.ServeHTTP(%q, %q) returned status %d, want %d", tt.method, tt.path, resp.StatusCode, tt.wantStatus)
            }
            bodyBytes, _ := io.ReadAll(resp.Body)

            resp.Body.Close()

            for _, want := range tt.wantBodyContains {
                if !strings.Contains(string(bodyBytes), want) {
                    t.Errorf("router.ServeHTTP(%q, %q) returned body %s, want body to contain %s", tt.method, tt.path, bodyBytes, want)
                }
            }

        })
    }
}
```

IN:

```sh
go test ./...
```

OUT:

```text
ok      gitlab.com/efronlicht/blog/articles/backendbasics/cmd/graduation        0.005s
```

Or to give a little more detail:

IN:

```sh
go test -v ./...  | RG "PASS"
```

OUT

```
--- PASS: TestNotFound (0.00s)
--- PASS: TestGraduation (0.00s)
    --- PASS: TestGraduation/GET/->200-OK (0.00s)
    --- PASS: TestGraduation/GET/panic->500-Internal-Server-Error (0.00s)
    --- PASS: TestGraduation/POST/greet/json->403-Forbidden (0.00s)
    --- PASS: TestGraduation/POST/greet/json->200-OK (0.00s)
    --- PASS: TestGraduation/GET/time?tz=America%2FNew_York->200-OK (0.00s)
    --- PASS: TestGraduation/GET/time?tz=invalid->400-Bad-Request (0.00s)
```

### Conclusion

That's it! Congratulations on reading all, uh

IN

```sh
wc -w backendbasics.md backendbasics2.md backendbasics3.md
```

OUT

```text
8152 backendbasics.md
5470 backendbasics2.md
10801 backendbasics3.md
24423 total
```

**24,423** words of this series so far.

With luck, you should feel significantly more comfortable with the basics of building backend infrastructure 'from scratch' (well, from the stdlib) in Go. This article is not even close to comprehensive. Beyond the fact we only barely scratched the surface of databases, there are many other topics we didn't cover, which include but are not limited to:

- authentication and authorization
- cryptography
- non-tcp transport-layer protocols (udp, unix sockets, etc)
- non-http application-layer protocols (grpc, thrift, etc)
- websockets
- foriegn function interfaces (FFI)
- graph databases
- distributed systems
- concurrency
- etc etc etc.

A brief personal note before we finish up: I've been blown away by the positive response to this article so far. Last time I checked, my reddit post about this article had nearly **60,000** views, well more than double the response to my [second-most-popular-article](https://eblog.fly.dev/quirks.html). Not bad, considering this website has no link-trading, no click-bait, no SEO, no ads, ~~ and most importantly ~~, no javascript - I've deliberately avoided doing any of the things you're "supposed" to do to get views on the internet, but an audience has found me anyway. Nice to know web 1.0 has a little life left in it yet.

A note on libraries and frameworks: there's nothing wrong with using someone else's code to solve a problem. I don't expect you to walk away from this article, climb on to the highest peak with nothing but a thin robe, a laptop, and a go compiler, and become completely self-sufficient. Libraries are just fine, and we've used a few in this article. My issue is with the attitude of reaching for a solution _without knowing the problem the solution is trying to solve_. Taking the effort to try and build something yourself is the only way to evaluate whether a library or framework is actually good or bad for your use case.

P.P.S: As I write this on **September 18, 2023** - Gophercon is here in San Diego next week! If you'd like to meet up, contact me on my email or linkedin and we can get some chinese food or something.

I'm going to stop this series here for now. The promised opinion-piece will come, _eventually_, but I might want to do some more technical material first. Also, frankly, I'm probably going to get a job, which will slow down the pace a bit.

Speaking of which...
> Like this article? Need help making great software, or just want to save a couple hundred thousand dollars on your cloud bill? **I am available on a contract or full-time basis**. Professional enquiries should be emailed to <efron.dev@gmail.com>, or contact me at <https://linkedin.com/in/efronlicht>.
