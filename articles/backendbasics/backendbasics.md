# Backend from the Beginning, Pt 1: Introduction, TCP, DNS, HTTP

A software article by Efron Licht

September 2023

<<article list placeholder>>

This article is part of a series on backend web development in Go. It will cover the basics of backend web development, from the ground up, without using a framework. It will cover the basics of TCP, DNS, HTTP, the `net/http` and `encoding/json` packages, middleware and routing. It should give you everything you need to get started writing professional backend web services in Go.

Source code for this article (& the entire blog) is publically available on my [gitlab](https://gitlab.com/efronlicht/blog).

## Introduction: when I hear the word 'framework' I reach for my gun

One of the most common questions I get from new developers starting with Go is **"what web framework should I use?"** The answer I always give is "you don't need a framework", but the problem is, backend devs are _used_ to frameworks.

Thinking about it, the motivation is understandable: engineers are under a lot of pressure, and the internet _seems_ really complicated, the idea of learning all these layers of abstraction (tcp, http, etc) is daunting, and _everyone else seems to use a framework_ - in most languages (javascript, python, etc), it's practically required. There's only one problem with this: **it means you never learn how things actually work**. Constantly relying on suites of specialized tools rather than learning the basics is the equivalent of being a senior chef who can't use a knife. Sure, you can argue that your fancy food processor chops faster, but the second you need to do something for which your pre-packaged tools aren't designed, you're screwed; you have no idea how to do it yourself and no time to learn.

This may sound like an exaggeration, but **I have now met four different senior software engineers who couldn't tell me how to make a HTTP request to google without a framework**.

For the record, you send this message to `142.250.189.14:80`:

```http
GET / HTTP/1.1
Host: google.com
```

It's five words.

If you don't know what stuff means or how I got that IP address, don't worry; we'll get to that. The point is, it's not actually that hard; hard is ten thousand layers of callbacks and libraries and frameworks and tools and languages and abstractions and indirections and wrappers around wrappers. The problem is, most software engineers are so used to having so many layers of abstraction between them and the network that 'getting to the bottom' seems impossible. Now, some may argue that this is OK, because the increased power & flexibility of frameworks leads to faster, better, software development. The only problem is, software isn't getting better; it's getting _measurably worse_. Both [desktop software](https://www.youtube.com/watch?v=GC-0tCy4P1U&t=2190s) and [web pages](https://httparchive.org/reports/state-of-the-web#bytesTotal) are measurably slower year over year. Software is getting slower faster than computers are speeding up. With this in mind, it's no surprise that software is **drowning in complexity**. Every year, we add more layers of abstraction, more libraries, more frameworks, more tools, more languages, more everything, but even our 'experts' don't understand the basics of the stuff they're building. It's no wonder that software is so slow and buggy and impossible to maintain.

> "An idiot admires complexity, a genius admires simplicity"

- Terry A. Davis

Of course, knowing that things are done badly doesn't help you learn how do do it _well_, so I'm writing this series of articles to try and fill the gap by teaching the basics of backend web development in Go. Each article will be filled with _real_ programs you can run on your computer, not cherry-picked code samples that don't even compile.

This series will not be enough to teach you everything. At best it will expose you to enough of how things _really_ work that you can start seeing the edges of your knowledge and begin filling in the gaps yourself. It will also necessarially have to simplify or omit some details or tell "white lies" to make things easier to understand; there's no substitute for experience (or for reading the source code & documentation of the standard library).  It will also largely omit databases; I hope to make those the subject of a future series.

That said, I hope it will help.

## Series overview

### 1. backend basics, part 1: TCP, DNS, & HTTP

- What is the internet? What problems does it solve? What's TCP/IP? How do computers talk to each other?
- What's DNS? How do we turn [www.google.com](https://www.google.com) into an IP address?
- What's HTTP and how does it work? How would we read or write a HTTP request or response by hand, without a library?
- Building a Request/Response library from scratch
  
### 2. Practical backend: `net/http` and `encoding/json`

In the second article, we'll graduate to using the `net/http` and `encoding/json` packages to build basic web clients and servers that can handle most day-to-day backend workloads. We'll start diving into Go's standard library and  the show how it provides everything you need for basic client/server HTTP communication; using `net/http` and `net/url` to send and receive HTTP requests and responses, `encoding/json` to manage our API payloads, and `context` to manage timeouts and cancellation.

### 3. Finishing touches: middlewares, routing, and basic database access

In the third article, we'll cover middleware and routing, the two 'missing pieces' of the `net/http` package. These are the bits that usually make people reach for a framework, but they're actually pretty simple to implement yourself. We'll also cover basic database access using the `database/sql` package.

### 4. When I hear the word 'framework' I reach for my gun

~~I'll talk a lot of shit on web frameworks, and we can bask in a smug feeling of self-satisfaction together.~~ I might not write this part; I'll only do it if I think I can make a point coherent rather than preach to the previously converted.

## What is backend?

'Backend' is connecting together computers via the internet. You know what a computer is, so...

### What's the internet anyways?

What is the internet? No, I'm serious. What is the problem the internet solves? The internet is a _network_ of computers that can reliably communicate with each other even if some of the computers 'in the middle' are down. It allows you to reliably send messages (that is, text or binary data) to other computers, even if you don't know where those computers are or how they're connected to you. You can send a message to another computer, so long as there is a path of computers from you (the `LOCALADDR`) to the destination computer (the `REMOTEADDR`)**.

To do this, the internet must solve two problems:

- How do I send a message to another computer, even if I don't have a direct connection to it? `ROUTING`
- How do I make sure that when I send a message through a network it gets to the right place, in order, and all of it gets through? `COHERENCE`
Both problems are solved by a protocol: the **I**nternet **P**rotocol (`IP`) solves `ROUTING`, and the **T**ransmission **C**ontrol **P**rotocol (`TCP`) solves `COHERENCE`. Together, they're called `TCP/IP`. That's how the internet works.

### TCP: how do I make sure that when I send a message through a network all of it gets through, in order?

The details of TCP are out of scope for this article, but at a high level it looks like this:

- You send packets of data to the remote computer. Each packet has a sequence number, ("which packet is this?") and a checksum ("did this packet get corrupted in transit?"). The remote computer sends back an acknowledgement ("I got packet 5") for each packet you send.
- If you don't get an acknowledgement for a packet, you resend it; if you get a corrupted packet, you resend it.
- This back-and-forth ensures that all of the data gets through, in order, and that you know when it doesn't.

### IP: how do I make sure that when I send a message through a network it gets to the right place?

IP is more complicated. The following explanation is wrong at pretty much every level if you zoom in enough, but it's a good enough approximation for our purposes:

- Each computer on the internet has an `address`, an identifier that tells other computers how to get to it. This address is called an `IP address`, or sometimes just an `IP`.
- They also have a list of other computers it knows about, and how to get to them. This list is called a **routing table**.
- When you send a message to another computer, your computer looks at its routing table to see if it knows how to get to that computer. If it does, it sends the message to the next computer in the chain. If it doesn't, it sends the message to the next computer in the chain that it _does_ know how to get to; they keep doing this until the message gets to the right computer.
- If there is no path from your computer to the destination computer, the message fails.

## Addresses & Ports

OK, so how do we actually send a message to another computer? We need to know two things: the `address` of the computer we want to send a message to, and the `port` of the service we want to send a message to.

## IP Addresses

IP Addresses come in two forms: `ipv4`, a 32-bit number; or `ipv6`, a 128-bit number. They look like this:

IPV4 looks like this: DDD.DDD.DDD.DDD, where DDD is a number between 0 and 255.

IPV6 looks like this: XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX:XXXX, where XXXX is a 16-bit hexadecimal number; that is, each X is one of `0..=9` or `a..=f`

| IP Address | Type | Note |
|------------|------|-------------|
| 192.168.000.001| ipv4 | localhost; refers to hosting computer |
| 192.168.0.1| ipv4 | same as above; you can omit leading zeroes |
| 0000:0000:0000:0000:0000:ffff:c0a8:0001 | ipv6 | refers to same computer as above; ipv4 addresses can be embedded in ipv6 addresses by prefixing them with `::ffff:` |
| ::ffff:c0a8:0001 | ipv6 | same as above; you can omit leading zeroes |
| 2a09:8280:1::a:791 | ipv6 | fly.io |

## Ports

It's common for a computer to want to host multiple internet services that behave in different ways. For example, we could want to host a game server (like `starcraft`), a web server (like this website), and a database (like `postgresql`) all on the same computer. Since they're all on the same physical computer, they'll share an IP address, so we'll need some way to tell apart requests to the file server from requests to the game server. We do this by assigning a `PORT` to each service. A port is just a number between 0 and 65535. Even if we're only hosting one service, each service needs (at least one) port.

`eblog` is hosted at port 6483. The following table lists default ports for some common services:

| Service | Port |
|---------|------|
| HTTP    | 80   |
| HTTPS   | 443  |
| SSH     | 22   |
| SMTP    | 25   |
| DNS     | 53   |
| FTP     | 21   |
| Postgres| 5432 |

## example 1: basic TCP/IP server

Let's build a basic TCP/IP server and client to demonstrate how this works. We'll build a server that listens on port 6483, and a client that connects to it. Anything sent on stdin on the client (that is, typed into the terminal) will be sent to the server, a line at a time. Anything line received on the server will be uppercased and sent back to the client.

That is, an example session might look like this:

```text
SERVER: (starts listening on port 6483)
CLIENT: (connects to server)
CLIENT: "hello, world!"
SERVER: "HELLO, WORLD!"
CLIENT: "goodbye, world!"
SERVER: "GOODBYE, WORLD!"
CLIENT: (disconnects)
```

To briefly review, the following functions and types are relevant to our examples:

|function/struct|description| implements
|---------------|-----------| ---------|
|`net.Listen`|listens for connections on a port|
|`net.Dial`|connects to a server at an IP address and port|
|`net.TCPConn` | bidirectional TCP connection| `io.Reader`, `io.Writer`, `net.Conn`
|`net.Conn`| bidirectional network connection| `io.Reader`, `io.Writer`
|`bufio.Scanner`|reads lines from a `io.Reader`|
|`fmt.Fprintf` | as `fmt.Printf`, but writes to a `io.Writer`|
| `flag.Int` | register an integer command line flag |
| `flag.Parse` | parse previously registered command line flags |
| `log.Printf` | as `fmt.Fprintf(os.Stderr, ...)`, but with a timestamp and newline |
| `log.Fatalf` | as `log.Printf`, but calls `os.Exit(1)` after printing |

- ### client

    Let's write the client first: we'll call it `writetcp`.

    ```go
    // writetcp connects to a TCP server at at localhost with the specified port (8080 by default) and forwards stdin to the server,
    // line-by-line, until EOF is reached.
    // received lines from the server are printed to stdout.
    package main

    import (
        "bufio"
        "flag"
        "fmt"
        "log"
        "net"
        "os"
    )

    func main() {
        const name = "writetcp"
        log.SetPrefix(name + "\t")

        // register the command-line flags: -p specifies the port to connect to
        port := flag.Int("p", 8080, "port to connect to")
        flag.Parse() // parse registered flags

        conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{Port: *port})
        if err != nil {
            log.Fatalf("error connecting to localhost:%d: %v", *port, err)
        }
        log.Printf("connected to %s: will forward stdin", conn.RemoteAddr())

        defer conn.Close()
        go func() { // spawn a goroutine to read incoming lines from the server and print them to stdout.
            // TCP is full-duplex, so we can read and write at the same time; we just need to spawn a goroutine to do the reading.

            for connScanner := bufio.NewScanner(conn); connScanner.Scan(); {

                fmt.Printf("%s\n", connScanner.Text()) // note: printf doesn't add a newline, so we need to add it ourselves

                if err := connScanner.Err(); err != nil {
                    log.Fatalf("error reading from %s: %v", conn.RemoteAddr(), err)
                }
                if connScanner.Err() != nil {
                    log.Fatalf("error reading from %s: %v", conn.RemoteAddr(), err)
                }
            }
        }()

        // read incoming lines from stdin and forward them to the server.
        for stdinScanner := bufio.NewScanner(os.Stdin); stdinScanner.Scan(); { // find the next newline in stdin
            log.Printf("sent: %s\n", stdinScanner.Text())
            if _, err := conn.Write(stdinScanner.Bytes()); err != nil { // scanner.Bytes() returns a slice of bytes up to but not including the next newline
                log.Fatalf("error writing to %s: %v", conn.RemoteAddr(), err)
            }
            if _, err := conn.Write([]byte("\n")); err != nil { // we need to add the newline back in
                log.Fatalf("error writing to %s: %v", conn.RemoteAddr(), err)
            }
            if stdinScanner.Err() != nil {
                log.Fatalf("error reading from %s: %v", conn.RemoteAddr(), err)
            }
        }

    }
    ```

- ### server

    Now let's put together the server; since it echoes back whatever it receives, in uppercase, we'll call it `tcpupperecho`.

    Usually when working in backend, we want to s.parate your 'business logic' from the networking code. Since all of go's networking APIs use the [net.Conn](https://golang.org/pkg/net/#Conn) interface, which implements both [io.Reader](https://golang.org/pkg/io/#Reader) and [io.Writer](https://golang.org/pkg/io/#Writer), we can write our business logic using standard text-handling functions and structs like [fmt.Fprintf](https://golang.org/pkg/fmt/#Fprintf) and [bufio.Scanner](https://golang.org/pkg/bufio/#Scanner).

    Our server's 'business logic' will look like this:

    ```go
    // echoUpper reads lines from r, uppercases them, and writes them to w.
    func echoUpper(w io.Writer, r io.Reader) {
        scanner := bufio.NewScanner(r)
        for scanner.Scan() {
            line := scanner.Text()
            // note that scanner.Text() strips the newline character from the end of the line,
            // so we need to add it back in when we write to w.
            fmt.Fprintf(w, "%s\n", strings.ToUpper(line))
        }
        if err := scanner.Err(); err != nil {
            log.Printf("error: %s", err)
        }
    }
    ```

    Which we can then use in our server like this:

    ```go

    // tcpupperecho serves tcp connections on port 8080, reading from each connection line-by-line and writing the upper-case version of each line back to the client.

    package main

    import (
        "bufio"
        "flag"
        "fmt"
        "io"
        "log"
        "net"
        "strings"
    )

    func main() {
        const name = "tcpupperecho"
        log.SetPrefix(name + "\t")

        // build the command-line interface; see https://golang.org/pkg/flag/ for details.
        port := flag.Int("p", 8080, "port to listen on")
        flag.Parse()

        // ListenTCP creates a TCP listener accepting connections on the given address.
        // TCPAddr represents the address of a TCP end point; it has an IP, Port, and Zone, all of which are optional.
        // Zone only matters for IPv6; we'll ignore it for now.
        // If we omit the IP, it means we are listening on all available IP addresses; if we omit the Port, it means we are listening on a random port.
        // We want to listen on a port specified by the user on the command-line.
        // see https://golang.org/pkg/net/#ListenTCP and https://golang.org/pkg/net/#Dial for details.
        listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: *port})
        if err != nil {
            panic(err)
        }
        defer listener.Close() // close the listener when we exit main()
        log.Printf("listening at localhost: %s", listener.Addr())
        for { // loop forever, accepting connections one at a time

            // Accept() blocks until a connection is made, then returns a Conn representing the connection.
            conn, err := listener.Accept()
            if err != nil {
                panic(err)
            }
            go echoUpper(conn, conn) // spawn a goroutine to handle the connection
        }
    }
    ```

- ### demonstration

    Let's try it out.  In one terminal, we'll run the server:

    IN

    ```sh
        go build -o tcpupperecho ./tcpupperecho.go
        ./tcpupperecho -p 8080 # run the server, listening on port 8080
        ```
    ```

    OUT:

    ```text
        tcpupperecho    2023/09/07 10:13:13 listening at localhost: [::]:8080
    ```

    Let's run the client in another terminal and send it a message:

    ```sh
        $ go build -o writetcp ./writetcp.go
        $ ./writetcp -p 8080 # run the client, connecting to localhost:8080
        > writetcp 2023/09/07 10:20:32 connected to 127.0.0.1:8080: will forward stdin
        hello
        writetcp        2023/09/07 10:20:49 sent: hello
        HELLO
    ```

    And checking in back in the server terminal, we see:

    ```sh
    tcpupperecho    2023/09/07 10:20:49 received: hello
    ```

### Connecting to a server on the internet

This works fine for local addresses, but what if we want to connect to a server on the internet? Most of the time, we don't know the `IP` address of the server we want to connect to; we only know its `domain name`, like `google.com` or `eblog.fly.dev`. How do we connect to a server at a domain name?

#### DNS

**D**omain **N**ame **S**ervice, or `DNS`, is a service that maps domain names to IP addresses. It's essentially a big table that looks like this:  

|domain| last known ipv4 | last known ipv6 |
|------|-----------------|-----------------|
| google.com | 142.250.217.142 | 2607:f8b0:4007:801::200e
| eblog.fly.dev |  66.241.125.53 | 2a09:8280:1::37:6bbc

There are multiple `DNS` providers. Your ISP usually provides one, and there are public ones like Google's, available at both `8.8.8.8` and `4.4.4.4`. (Since you can't resolve a domain name without knowing the IP address of a DNS server, you need to know at least one IP 'by heart' to get started.)

Browsers and other clients use `DNS` service to look up the IP address of a domain name.

### Finding the IP address of a server

2021/08/18 16:00:00 tcpupperecho listening at localhost:
OK, so we want to connect to a server at a **web address**: say, `https://eblog.fly.dev`. How do we do that? Well, first we need to get the IP address of the server. The **domain name service**, or `DNS`, is a service that maps domain names to IP addresses. You can use the built-in `nslookup` command to look up the IP address of a domain name from your command-line on windows, mac, or linux.

IN:

```bash
nslookup eblog.fly.dev
```

OUT:

```text
Server:  UnKnown
Address:  192.168.1.1

Non-authoritative answer:
Name:    eblog.fly.dev
Addresses:  2a09:8280:1::37:6bbc
          66.241.125.53
```

Within a Go program, use [`net.LookupIP`](https://golang.org/pkg/net/#LookupIP) to look up the IP address or addresses of a domain name. The following full program duplicates the functionality of `nslookup`:

```go
// dns is a simple command line tool to lookup the ip address of a host;
// it prints the first ipv4 and ipv6 addresses it finds, or "none" if none are found.
package main

import (
    "fmt"
    "log"
    "net"
    "os"
)

func main() {
    if len(os.Args) != 2 {
        log.Infof("%s: usage: <host>", os.Args[0])
        log.Fatalf("expected exactly one argument; got %d", len(os.Args)-1)
    }
    host := os.Args[1]
    ips, err := net.LookupIP(host)
    if err != nil {
        log.Fatalf("lookup ip: %s: %v", host, err)
    }
    if len(ips) == 0 {
        log.Fatalf("no ips found for %s", host) // this should never happen, but just in case
    }
    // print the first ipv4 we find
    for _, ip := range ips {
        if ip.To4() != nil {
            fmt.Println(ip)
            goto IPV6 // goto considered awesome
        }
    }
    fmt.Printf("none\n") // only print "none" if we don't find any ipv4 addresses

IPV6: // print the first ipv6 we find
    for _, ip := range ips {
        if ip.To4() == nil {
            fmt.Println(ip) // we don't need to check for nil here, since we know we have at least one ip address
            return
        }
    }
    fmt.Printf("none\n")
}

```

IN:

```bash
go build -o dns ./dns.go # build the dns command
./dns eblog.fly.dev # run the dns command
```

OUT:

```text
66.241.125.53
2a09:8280:1::37:6bbc
```

### putting it together: `DNS` & `HTTP`

We now have everything we need to for the basics of internet browsing: we can look up the IP address of a domain name, and we can connect to a server at an IP address and port.

When you type a URL into your browser, it does the following:

- looks up the IP address of the domain name
- connects to the server at that IP address and port
- sends a `HTTP` request to the server

But wait, what's HTTP? The **H**yper**T**ext **T**ransfer **P**rotocol is a **text-based** protocol for sending messages over the internet. It's honestly incredibly simple:

### HTTP Requests

A HTTP Request is plain text, and looks like this:

```http
<METHOD>  <PATH>  <PROTOCOL/VERSION>
Host: <HOST>
[<HEADER>: <VALUE>]
[<HEADER>: <VALUE>]
[<HEADER>: <VALUE>] (these guys are optional)

[<REQUEST BODY>] (this is also optional).
```

To give a more concrete example, the most basic HTTP request you could send to get this webpage would look like this:

```http
GET /backendbasics.html HTTP/1.1
Host: eblog.fly.dev
```

(A couple of gotchas here: the line breaks are windows-style `\r\n`, not unix-style `\n`; and the request must end with a blank line.)

Let's break this down. We can read this as

- **GET** the resource on the host `eblog.fly.dev`
- at the path `/backendbasics.html`
- using the **HTTP/1.1** protocol.

The first line is the **REQUEST LINE**. It has three parts:

- the **METHOD** (like `GET`, `POST`, `PUT`, `DELETE`, etc) tells the server what kind of request this is. For now, we only care about two: `GET` means "READ", `POST` means "WRITE".
- the **PATH** is the path to the resource you want to access; this is the part after `.com` or `.dev` in a web address. Here, the **PATH** is `/backendbasics.html`
- the **PROTOCOL/VERSION** is the protocol and version of the request; almost always `HTTP/1.1` or `HTTP/2.0`

The **REQUEST LINE** is followed by one or more **HEADERS**. A **HEADER** is a key-value pair, separated by a colon (`:`). The **key** should be formatted in `Title-Case`, and the **value** should be formatted in `lower-case`; for example, `Content-Type: application/json`. A few headers have official meanings in the HTTP spec, but most are just suggestions to the server about how to handle the request. Technically, headers are [MIME](https://en.wikipedia.org/wiki/MIME) headers, but we have enough acronyms to deal with already; we'll just call them headers for now.

The **HOST** header is required; it tells the server which domain name you're trying to access. For this article, the **HOST** header is `Host: eblog.fly.dev`  Other headers are optional, and can be used to send additional information to the server. Some common headers include:

| header | description | example(s) |
|--------|-------------|------|
| `Accept-Encoding`| I can accept responses encoded with these encodings | `gzip`, `deflate` |
| `Accept` | the types of responses the client can accept | `text/html` |
| `Cache-Control` | how the client wants the server to cache the response | `no-cache` |
| `Content-Encoding`| my response body is encoded using: | `gzip`, `deflate` |
| `Content-Length` | my body is N bytes long | 47
| `Content-Type` | the type of the request body | `application/json` |
| `Date` | the date and time of the request | `Tue, 17 Aug 2021 23:00:00 GMT` |
| `Host` | the domain name of the server you're trying to access | `eblog.fly.dev` |
| `User-Agent` | the name and version of the client making the request | `curl/7.64.1`, `Mozilla/5.0 (Linux; Android 8.0.0; SM-G955U Build/R16NW)` |

Your browser sends a lot more headers than this: you can see them by opening the developer tools and looking at the network tab.

Here's what chrome sent when I opened this page on the devtools network tab (that is, when I sent a `GET` request to `https://eblog.fly.dev/backendbasics.html`):

```http
GET / HTTP/1.1
Host: eblog.fly.dev
Accept-Encoding: gzip, deflate, br
Accept-Language: en-US,en;q=0.9
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7
Cache-Control: no-cache
Pragma: no-cache
Sec-Ch-Ua-Mobile: ?1
Sec-Ch-Ua-Platform: "Android"
Sec-Ch-Ua: "Chromium";v="116", "Not)A;Brand";v="24", "Google Chrome";v="116"
Sec-Fetch-Dest: document
Sec-Fetch-Mode: navigate
Sec-Fetch-Site: none
Sec-Fetch-User: ?1
Upgrade-Insecure-Requests: 1
User-Agent: Mozilla/5.0 (Linux; Android 8.0.0; SM-G955U Build/R16NW) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.141 Mobile Safari/537.36
```

It's OK to have multiple headers with the same key; for example, you might have multiple `Accept-Encoding` headers, each with a different encoding. Alternatively, you can separate multiple values with a comma.
That is, these two SHOULD be equivalent:

```http
Accept-Encoding: gzip
Accept-Encoding: deflate
```

and

```http
Accept-Encoding: gzip, deflate
```

The server will usually pick the first one it understands. Servers are _supposed_ to treat the keys as case-insensitive, but in practice this is not always the case; similarly, some web servers don't properly handle multiple headers with the same key.

#### URL-encoding

Note that the HTTP request separates it's sections using the following characters: `' '`, `'\r'`, `'\n'`, `:`. That means we couldn't use these characters in the request line or in headers without confusing the server; it wouldn't know if we were trying to separate a section or give it literal text.

As such, URL paths and headers can't contain these characters; we must 'escape' them using [URL %-encoding](https://en.wikipedia.org/wiki/Percent-encoding) before sending them to the server. URL-encoding is actually pretty simple: take any ASCII character, and turn it into it's value in hexadecimal, prefixed with a `%`. For example, the space character is `0x20` in hexadecimal, so we encode it as `%20`. The percent character itself is `0x25`, so we encode it as `%25`.

The following characters can ALWAYS be used in a URL path or header without escaping:

|category|characters|
|--------|----------|
|lowercase ascii letters|`abcdefghijklmnopqrstuvwxyz`|
|uppercase ascii letters|`ABCDEFGHIJKLMNOPQRSTUVWXYZ`|
|digits|`0123456789`|
|unreserved characters|`-._~`|
| escaped| `%` followed by two hexadecimal digits|

But some characters can only be used unescaped in certain contexts:

| character set | context | note |
|---------------|---------|------|
| `:/?#[]@` | path | i've never seen `[]`; `@` is for authentication |
| `&` | query parameter | separates query parameters |
| `+` | query parameter | used to encode spaces in query parameters |
| `=` | query parameter | separates keys from values in query parameters |
| `;` | path | separates path segments; rarely used |
| `$` | path | rarely used |

Everything else must be escaped. For example, the following request path is valid:

```http
GET /backendbasics.html HTTP/1.1
Host: eblog.fly.dev
```

but this one isn't:

```http
GET /backend basics.html HTTP/1.1
Host: eblog.fly.dev
```

And should be encoded as:

```http
GET /backend%20basics.html HTTP/1.1
Host: eblog.fly.dev
```

The [`url.PathEscape`](https://golang.org/pkg/net/url/#PathEscape) and [`url.PathUnescape`](https://golang.org/pkg/net/url/#PathUnescape) functions in the standard library can be used to escape and unescape a string for use in a URL path or header; we'll cover that package in more detail in a later article.

#### Query Parameters

The PATH can also contain **query parameters**; these are key-value pairs in the form `key=value` that come after that path. You end the 'normal' part of the path with a `?`, and then add the query parameters, separating each with a `&`.

If I want to make a google search for "backend_basics", I would send the following request:

```http
GET /search?q=backendbasics HTTP/1.1
Host: google.com
```

This has a single query parameter, with **KEY** `q` and VALUE `backendbasics`. I could add additional query parameters by separating them with `&`:

The **[scryfall](https://scryfall.com/)** API allows you to search for magic cards using a variety of query parameters: if I wanted to search for cards with the word "ice" in their name, ordered by their release date, I would send the following request:

```http
GET /search?q=ice&order=released&dir=asc HTTP/1.1
```

This would have three query parameters: "q=ice", "order=released", and "dir=asc". Note that the `=` and `&` characters are not escaped in the query parameters.

That's pretty much all there is to HTTP requests. Let's try sending a `HTTP` request to `eblog.fly.dev` using TCP. The following complete program, `sendreq`, sends a HTTP request to a server at a given host, port, and path, and prints the response to stdout.

```go
// sendreq sends a request to the specified host, port, and path, and prints the response to stdout.
// flags: -host, -port, -path, -method
package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "net"
    "os"
    "strings"
)

// define flags
var (
    host, path, method string
    port               int
)

func main() {
    // initialize & parse flags
    flag.StringVar(&method, "method", "GET", "HTTP method to use")
    flag.StringVar(&host, "host", "localhost", "host to connect to")
    flag.IntVar(&port, "port", 8080, "port to connect to")
    flag.StringVar(&path, "path", "/", "path to request")
    flag.Parse()

    // ResolveTCPAddr is a slightly more convenient way of creating a TCPAddr.
    // now that we know how to do it by hand using net.LookupIP, we can use this instead.
    ip, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
    if err != nil {
        panic(err)
    }

    // dial the remote host using the TCPAddr we just created...
    conn, err := net.DialTCP("tcp", nil, ip)
    if err != nil {
        panic(err)
    }

    log.Printf("connected to %s (@ %s)", host, conn.RemoteAddr())

    defer conn.Close()

    var reqfields = []string{
        fmt.Sprintf("%s %s HTTP/1.1", method, path),
        "Host: " + host,
        "User-Agent: httpget",
        "", // empty line to terminate the headers

        // body would go here, if we had one
    }
    // e.g, for a request to http://eblog.fly.dev/
    // GET / HTTP/1.1
    // Host: eblog.fly.dev
    // User-Agent: httpget
    //

    request := strings.Join(reqfields, "\r\n") + "\r\n" // note windows-style line endings

    conn.Write([]byte(request))
    log.Printf("sent request:\n%s", request)

    for scanner := bufio.NewScanner(conn); scanner.Scan(); {
        line := scanner.Bytes()
        if _, err := fmt.Fprintf(os.Stdout, "%s\n", line); err != nil {
            log.Printf("error writing to connection: %s", err)
        }
        if scanner.Err() != nil {
            log.Printf("error reading from connection: %s", err)
            return
        }
    }

}
```

Let's try it out on the index page of this blog (running on localhost:8080):

```go
go build -o sendreq ./sendreq.go    
./sendreq -host eblog.fly.dev -port 8080
```

```
2023/09/07 13:59:19 connected to localhost (@ 127.0.0.1:8080)
2023/09/07 13:59:19 sent request:
```

```http
GET / HTTP/1.1
Host: localhost
User-Agent: httpget
```

And we get back a response, a _redirect_ to `/index.html`:

```http
HTTP/1.1 308 Permanent Redirect
Content-Type: text/html; charset=utf-8
E-Req-Id: b641130b240142ae82ae8b122c35c80f
E-Trace-Id: 086e9e55-364b-4cfd-b8fe-6497214af367
Location: /index.html
Date: Thu, 07 Sep 2023 20:59:19 GMT
Content-Length: 47

<a href="/index.html">Permanent Redirect</a>.
```

Let's quickly discuss the response format before we move on to the next section.

A **HTTP** response is also plain text, and looks like this:

```http
<PROTOCOL/VERSION> <STATUS CODE> <STATUS MESSAGE>
[<HEADER>: <VALUE>] (these guys are optional)
[<HEADER>: <VALUE>]
[<HEADER>: <VALUE>]

[<RESPONSE BODY>] (this is optional).
```

The first line is the **STATUS LINE**. It has three parts:

- the **PROTOCOL/VERSION** is the protocol and version of the response; it should always be the same as the request.
- the **STATUS CODE**  is a three-digit number that tells you whether the request succeeded or failed. The first digit tells you the general category of the response::
  - `1xx` means "informational". These are not used very often.
  - `2xx` means "success"  **200 OK** and **201** created are the only ones you'll see in practice.
  - `3xx` means "redirect" **301** and **308** are the only ones you'll see in practice.
  - `4xx` means "client error"; you're probably familiar with **404 Not Found** and **403 Forbidden**, but there are a lot of others.
  - `5xx` means "server error". **500 Internal Server Error** is the only one you'll see in practice; it's the default error code for any unhandled error.
Each status code has exactly one corresponding **STATUS MESSAGE**; for example, **200 OK** or **404 Not Found**. The status message is just a human-readable description of the status code.

Headers work the same way as in requests: they're key-value pairs separated by a colon (`:`), and the key should be formatted in `Title-Case` and the value should be formatted in `lower-case`. Headers of the response are usually 'symmetric' to the headers of the request; if you send an `Accept-Encoding: gzip` header, you'll usually get a `Content-Encoding: gzip` header back.

The final section is the **response body**. Here, it's some **HTML** that tells the browser to redirect to `/index.html`.  Don't worry, I'm not going to cover HTML: this is a backend article, not a frontend article.

Let's follow the redirect and request `/index.html`:

```sh
./sendreq -host eblog.fly.dev -port 8080 -path /index.html
```

We get back a 200 OK response and the (very sparse) contents of the index page:

```http
HTTP/1.1 200 OK
E-Req-Id: 47cf0abba4fd4629a9a926769649f653
E-Trace-Id: dc2c9528-0322-4a16-8688-8ce760fff374
Date: Thu, 07 Sep 2023 21:04:28 GMT
Content-Length: 1300
Content-Type: text/html; charset=utf-8

<!DOCTYPE html><html><head>
        <title>index.html</title>
        <meta charset="utf-8"/>
        <link rel="stylesheet" type="text/css" href="/dark.css"/>
        </head>
        <body>
        <h1> articles </h1>
<h4><a href="/performanceanxiety.html">performanceanxiety.html</a>
</h4><h4><a href="/onoff.html">onoff.html</a>
</h4><h4><a href="/fastdocker.html">fastdocker.html</a>
</h4><h4><a href="/README.html">README.html</a>
</h4><h4><a href="/mermaid_test.html">mermaid_test.html</a>
</h4><h4><a href="/quirks3.html">quirks3.html</a>
</h4><h4><a href="/console-autocomplete.html">console-autocomplete.html</a>
</h4><h4><a href="/console.html">console.html</a>
</h4><h4><a href="/cheatsheet.html">cheatsheet.html</a>
</h4><h4><a href="/testfast.html">testfast.html</a>
</h4><h4><a href="/quirks2.html">quirks2.html</a>
</h4><h4><a href="/bytehacking.html">bytehacking.html</a>
</h4><h4><a href="/benchmark_results.html">benchmark_results.html</a>
</h4><h4><a href="/index.html">index.html</a>
</h4><h4><a href="/noframework.html">noframework.html</a>
</h4><h4><a href="/faststack.html">faststack.html</a>
</h4><h4><a href="/backendbasics.html">backendbasics.html</a>
</h4><h4><a href="/startfast.html">startfast.html</a>
</h4><h4><a href="/quirks.html">quirks.html</a>
</h4><h4><a href="/reflect.html">reflect.html</a>
```

### Dealing Programmatically with HTTP Requests and Responses

This is OK, but dealing with raw HTTP requests and Responses is kind of a pain. Before we dive into Go's `net/http` package, let's think about how we might implement a HTTP library ourselves.

We'd like a way to write our requests and responses without having to worry about the details of the protocol, like making sure our newlines are windows-style `\r\n` instead of unix-style `\n` or title-casing our headers.

That is, we'll need four things, roughly in order of difficulty (easiest first):

- a way to represent a HTTP request or response in memory
- a way to add headers to a request or response.
- a way to serialize them as text in the HTTP format
- a way to parse them from text in the HTTP format

#### The Request and Response structs

Restricting ourselves for now to HTTP 1.1, we can think of a HTTP request as a struct with the following fields:

```go

// Header represents a HTTP header. A HTTP header is a key-value pair, separated by a colon (:);
// the key should be formatted in Title-Case.
// Use Request.AddHeader() or Response.AddHeader() to add headers to a request or response and guarantee title-casing of the key.
type Header struct {Key, Value string}
// Request represents a HTTP 1.1 request.
type Request struct {
    Method string // e.g, GET, POST, PUT, DELETE
    Path string // e.g, /index.html
    Headers []struct {Key, Value string}  // e.g, Host: eblog.fly.dev
    Body string // e.g, <html><body><h1>hello, world!</h1></body></html>
}
```

and a HTTP response as a struct with the following fields:

```go
type Response struct {
    StatusCode int // e.g, 200
    Headers []struct {Key, Value string}  // e.g, Content-Type: text/html
    Body string // e.g, <html><body><h1>hello, world!</h1></body></html>
}
```

The following functions will build a request or response for us:

```go
func NewRequest(method, path, host, body string) (*Request, error) {
    switch {
    case method == "":
        return nil, errors.New("missing required argument: method")
    case path == "":
        return nil, errors.New("missing required argument: path")
    case !strings.HasPrefix(path, "/"):
        return nil, errors.New("path must start with /")
    case host == "":
        return nil, errors.New("missing required argument: host")
    default:
        headers := make([]Header, 2)
        headers[0] = Header{"Host", host}
        if body != ""  {
            headers = append(headers, Header{"Content-Length", fmt.Sprintf("%d", len(body))})
        }
        return &Request{Method: method, Path: path, Headers: headers, Body: body}, nil
    }
}

func NewResponse(status int, body string) (*Response, error) {
    switch {
    case status < 100 || status > 599:
        return nil, errors.New("invalid status code")
    default:
        if body == "" {
            body = http.StatusText(status)
        }
        headers := []Header {"Content-Length", fmt.Sprintf("%d", len(body))}
        return &Response{
            StatusCode: status,
            Headers: headers,
            Body: body,
        }, nil
    }
}
```

#### Adding headers

We'd like to be able to add headers to a request or response without worrying about casing of the keys. We'll do this with a 'builder' method on `*Request` and `*Response`:

```go
func (resp *Response) WithHeader(key, value string) *Response {
    resp.Headers = append(resp.Headers, Header{AsTitle(key), value})
    return resp
}
func (r *Request) WithHeader(key, value string) *Request {
    r.Headers = append(r.Headers, Header{AsTitle(key), value})
    return r
}
```

We can use these to build a request a header at a time:

```go
req, err := NewRequest("POST", "/api/v1/users", "eblog.fly.dev", `{"name": "eblog", "email": "efron.dev@gmail.com"}`)
if err != nil {
    panic(err)
}
req = req.WithHeader("Content-Type", "application/json").
    WithHeader("Accept", "application/json").
    WithHeader("User-Agent", "httpget")
```

But how is `AsTitle` implemented? Let's write a quick test first to make sure we understand the requirements:

```go
func TestTitleCaseKey(t *testing.T) {
    for input, want := range map[string]string{
        "foo-bar":      "Foo-Bar",
        "cONTEnt-tYPE": "Content-Type",
        "host":         "Host",
        "host-":        "Host-",
        "ha22-o3st":    "Ha22-O3st",
    } {
        if got := AsTitle(input); got != want {
            t.Errorf("TitleCaseKey(%q) = %q, want %q", input, got, want)
        }
    }
}
```

[MIME]headers are assumed to be ASCII-only, so we don't need to worry about unicode here.

```go
// AsTitle returns the given header key as title case; e.g. "content-type" -> "Content-Type"
// It will panic if the key is empty.
func AsTitle(key string) string {
    /* design note --- an empty string could be considered 'in title case', 
    but in practice it's probably programmer error. rather than guess, we'll panic.
    */
    if key == "" {
        panic("empty header key")
    }
    if isTitleCase(key) {
        return key
    }
    /* ---- design note: allocation is very expensive, while iteration through strings is very cheap.
    in general, better to check twice rather than allocate once. ----
    */
    return newTitleCase(key)
}



// newTitleCase returns the given header key as title case; e.g. "content-type" -> "Content-Type";
// it always allocates a new string.
func newTitleCase(key string) string {
    var b strings.Builder
    b.Grow(len(key))
    for i := range key {

        if i == 0 || key[i-1] == '-' {
            b.WriteByte(upper(key[i]))
        } else {
            b.WriteByte(lower(key[i]))
        }
    }
    return b.String()
}


// straight from K&R C, 2nd edition, page 43. some classics never go out of style.
func lower(c byte) byte {
    /* if you're having trouble understanding this:
        the idea is as follows: A..=Z are 65..=90, and a..=z are 97..=122.
        so upper-case letters are 32 less than their lower-case counterparts (or 'a'-'A' == 32).
        rather than using the 'magic' number 32, we use 'a'-'A' to get the same result.
    */
    if c >= 'A' && c <= 'Z' {
        return c + 'a' - 'A'
    }
    return c
}
func upper(c byte) byte {
    if c >= 'a' && c <= 'z' {
        return c + 'A' - 'a'
    }
    return c
}



// isTitleCase returns true if the given header key is already title case; i.e, it is of the form "Content-Type" or "Content-Length", "Some-Odd-Header", etc.
func isTitleCase(key string) bool {
    // check if this is already title case.
    for i := range key {
        if i == 0 || key[i-1] == '-' {
            if key[i] >= 'a' && key[i] <= 'z' {
                return false
            }
        } else if key[i] >= 'A' && key[i] <= 'Z' {
            return false
        }
    }
    return true
}

```

We run the test and it passes, so we're good to go. Compare to the actual standard library's [implementation](https://cs.opensource.google/go/go/+/refs/tags/go1.21.1:src/net/textproto/reader.go;l=632) of [`textproto.CanonicalMIMEHeaderKey`](https://golang.org/pkg/net/textproto/#CanonicalMIMEHeaderKey); ours is essentially the same but doesn't handle some corner cases and optimizations for common headers.

We'll implement the [`io.WriterTo`](https://golang.org/pkg/io/#WriterTo) interface on both of these structs so we can efficiently write them to a `net.Conn` or other `io.Writer`.

```go
// Write writes the Request to the given io.Writer.
func (r *Request) WriteTo(w io.Writer) (n int64, err error) {
    // write & count bytes written.
    // using small closures like this to cut down on repetition
    // can be nice; but you sometimes pay a performance penalty.
    printf := func(format string, args ...any) error {
        m, err := fmt.Fprintf(w, format, args...)
        n += int64(m)
        return err
    }
    // remember, a HTTP request looks like this:
    // <METHOD>  <PATH>  <PROTOCOL/VERSION>
    // <HEADER>: <VALUE>
    // <HEADER>: <VALUE>
    // 
    // <REQUEST BODY>

    // write the request line: like "GET /index.html HTTP/1.1"
    if err := printf("%s %s HTTP/1.1\r\n", r.Method, r.Path); err != nil {
        return n, err
    }

    // write the headers. we don't do anything to order them or combine/merge duplicate headers; this is just an example.
    for _, h := range r.Headers {
        if err := printf("%s: %s\r\n", h.Key, h.Value); err != nil {
            return n, err
        }
    }
    printf("\r\n") // write the empty line that separates the headers from the body
    err = printf("%s\r\n", r.Body) // write the body and terminate with a newline
    return n, err
}
```

Response has a nearly identical implementation:

```go
func (resp *Response) WriteTo(w io.Writer) (n int64, err error) {
    printf := func(format string, args ...any) error {
        m, err := fmt.Fprintf(w, format, args...)
        n += int64(m)
        return err
    }
    if err := printf("HTTP/1.1 %d %s\r\n", resp.StatusCode, http.StatusText(resp.StatusCode)); err != nil {
        return n, err
    }
    for _, h := range resp.Headers {
        if err := printf("%s: %s\r\n", h.Key, h.Value); err != nil {
            return n, err
        }

    }
    if err := printf("\r\n%s\r\n", resp.Body); err != nil {
        return n, err
    }
    return n, nil
}
```

#### Sidenote: Go's standard interfaces

Go has a number of standard interfaces that are used throughout the standard library. You've probably already seen [`io.Reader`](https://golang.org/pkg/io/#Reader) and [`io.Writer`](https://golang.org/pkg/io/#Writer), but there are a lot more. Many functions in the standard library work better with types that implement these interfaces; for example, [`io.Copy`](https://golang.org/pkg/io/#Copy) will copy from an `io.Reader` to an `io.Writer`, but if the `src` implements `[io.WriterTo](https://golang.org/pkg/io/#WriterTo)` or the `dst` implements [`io.ReaderFrom`](https://golang.org/pkg/io/#ReaderFrom), it will use those methods instead, which can be more efficient.

Similarly, [`fmt.Stringer`](https://golang.org/pkg/fmt/#Stringer) is used to get a string representation of a type, and [`encoding.TextMarshaler`](https://golang.org/pkg/encoding/#TextMarshaler) is used to get a byte slice representation of a type in order to serialize it out across the network or to disk.

We'll implement both of those interfaces on our `Request` and `Response` types for convenience and to make our tests easier to write.

All we need to do is call `WriteTo` and return the result:

```go
var _, _ fmt.Stringer = (*Request)(nil), (*Response)(nil) // compile-time check that Request and Response implement fmt.Stringer
var _, _ encoding.TextMarshaler = (*Request)(nil), (*Response)(nil)
func (r *Request) String() string { b := new(strings.Builder); r.WriteTo(b); return b.String() }
func (resp *Response) String() string { b := new(strings.Builder); resp.WriteTo(b); return b.String() }
func (r *Request) MarshalText() ([]byte, error) { b := new(bytes.Buffer); r.WriteTo(b); return b.Bytes(), nil }
func (resp *Response) MarshalText() ([]byte, error) { b := new(bytes.Buffer); resp.WriteTo(b); return b.Bytes(), nil }
```

#### Parsing HTTP Requests and Responses

One last thing: we'd like to be able to parse HTTP requests and responses from text. This is a bit more complicated than writing them, but given what we've done so far, it should be relatively straightforward.

```go
// ParseRequest parses a HTTP request from the given text.
func ParseRequest(raw string) (r Request, err error) {
    // request has three parts:
    // 1. Request linedd
    // 2. Headers
    // 3. Body (optional)
    lines := splitLines(raw)

    log.Println(lines)
    if len(lines) < 3 {
        return Request{}, fmt.Errorf("malformed request: should have at least 3 lines")
    }
    // First line is special.
    first := strings.Fields(lines[0])
    r.Method, r.Path = first[0], first[1]
    if !strings.HasPrefix(r.Path, "/") {
        return Request{}, fmt.Errorf("malformed request: path should start with /")
    }
    if !strings.Contains(first[2], "HTTP") {
        return Request{}, fmt.Errorf("malformed request: first line should contain HTTP version")
    }
    var foundhost bool
    var bodyStart int
    // then we have headers, up until the an empty line.
    for i := 1; i < len(lines); i++ {
        if lines[i] == "" { // empty line
            bodyStart = i + 1
            break
        }
        key, val, ok := strings.Cut(lines[i], ": ")
        if !ok {
            return Request{}, fmt.Errorf("malformed request: header %q should be of form 'key: value'", lines[i])
        }
        if key == "Host" { // special case: host header is required.
            foundhost = true
        }
        key = AsTitle(key)

        r.Headers = append(r.Headers, Header{key, val})
    }
    end := len(lines) - 1 // recombine the body using normal newlines; skip the last empty line.
    r.Body = strings.Join(lines[bodyStart:end], "\r\n")
    if !foundhost {
        return Request{}, fmt.Errorf("malformed request: missing Host header")
    }
    return r, nil
}


// ParseResponse parses the given HTTP/1.1 response string into the Response. It returns an error if the Response is invalid,
// - not a valid integer
// - invalid status code
// - missing status text
// - invalid headers
// it doesn't properly handle multi-line headers, headers with multiple values, or html-encoding, etc.zzs
func ParseResponse(raw string) (resp *Response, err error) {
    // response has three parts:
    // 1. Response line
    // 2. Headers
    // 3. Body (optional)
    lines := splitLines(raw)
    log.Println(lines)

    // First line is special.
    first := strings.SplitN(lines[0], " ", 3)
    if !strings.Contains(first[0], "HTTP") {
        return nil, fmt.Errorf("malformed response: first line should contain HTTP version")
    }
    resp = new(Response)
    resp.StatusCode, err = strconv.Atoi(first[1])
    if err != nil {
        return nil, fmt.Errorf("malformed response: expected status code to be an integer, got %q", first[1])
    }
    if first[2] == "" || http.StatusText(resp.StatusCode) != first[2] {
        log.Printf("missing or incorrect status text for status code %d: expected %q, but got %q", resp.StatusCode, http.StatusText(resp.StatusCode), first[2])
    }
    var bodyStart int
    // then we have headers, up until the an empty line.
    for i := 1; i < len(lines); i++ {
        log.Println(i, lines[i])
        if lines[i] == "" { // empty line
            bodyStart = i + 1
            break
        }
        key, val, ok := strings.Cut(lines[i], ": ")
        if !ok {
            return nil, fmt.Errorf("malformed response: header %q should be of form 'key: value'", lines[i])
        }
        key = AsTitle(key)
        resp.Headers = append(resp.Headers, Header{key, val})
    }
    resp.Body = strings.TrimSpace(strings.Join(lines[bodyStart:], "\r\n")) // recombine the body using normal newlines.
    return resp, nil
}
// splitLines on the "\r\n" sequence; multiple separators in a row are NOT collapsed.
func splitLines(s string) []string {
    if s == "" {
        return nil
    }
    var lines []string
    i := 0
    for {
        j := strings.Index(s[i:], "\r\n")
        if j == -1 {
            lines = append(lines, s[i:])
            return lines
        }
        lines = append(lines, s[i:i+j]) // up to but not including the \r\n
        i += j + 2 // skip the \r\n
    }
}
```

As before, let's write a few quick tests to make sure we understand the requirements.

I'm omitting the error cases for brevity here; this article is more than long enough already.

```go
func TestHTTPResponse(t *testing.T) {
    for name, tt := range map[string]struct {
        input string
        want  *Response
    }{
        "200 OK (no body)": {
            input: "HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n",
            want: &Response{
                StatusCode: 200,
                Headers: []Header{
                    {"Content-Length", "0"},
                },
            },
        },
        "404 Not Found (w/ body)": {
            input: "HTTP/1.1 404 Not Found\r\nContent-Length: 11\r\n\r\nHello World\r\n",
            want: &Response{
                StatusCode: 404,
                Headers: []Header{
                    {"Content-Length", "11"},
                },
                Body: "Hello World",
            },
        },
    } {
        t.Run(name, func(t *testing.T) {
            got, err := ParseResponse(tt.input)
            if err != nil {
                t.Errorf("ParseResponse(%q) returned error: %v", tt.input, err)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseResponse(%q) = %#+v, want %#+v", tt.input, got, tt.want)
            }

            if got2, err := ParseResponse(got.String()); err != nil {
                t.Errorf("ParseResponse(%q) returned error: %v", got.String(), err)
            } else if !reflect.DeepEqual(got2, got) {
                t.Errorf("ParseResponse(%q) = %#+v, want %#+v", got.String(), got2, got)
            }

        })
    }
}

func TestHTTPRequest(t *testing.T) {
    for name, tt := range map[string]struct {
        input string
        want  Request
    }{
        "GET (no body)": {
            input: "GET / HTTP/1.1\r\nHost: www.example.com\r\n\r\n",
            want: Request{
                Method: "GET",
                Path:   "/",
                Headers: []Header{
                    {"Host", "www.example.com"},
                },
            },
        },
        "POST (w/ body)": {
            input: "POST / HTTP/1.1\r\nHost: www.example.com\r\nContent-Length: 11\r\n\r\nHello World\r\n",
            want: Request{
                Method: "POST",
                Path:   "/",
                Headers: []Header{
                    {"Host", "www.example.com"},
                    {"Content-Length", "11"},
                },
                Body: "Hello World",
            },
        },
    } {
        t.Run(name, func(t *testing.T) {
            got, err := ParseRequest(tt.input)
            if err != nil {
                t.Errorf("ParseRequest(%q) returned error: %v", tt.input, err)
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseRequest(%q) = %#+v, want %#+v", tt.input, got, tt.want)
            }
            // test that the request can be written to a string and parsed back into the same request.
            got2, err := ParseRequest(got.String())
            if err != nil {
                t.Errorf("ParseRequest(%q) returned error: %v", got.String(), err)
            }
            if !reflect.DeepEqual(got, got2) {
                t.Errorf("ParseRequest(%q) = %+v, want %+v", got.String(), got2, got)
            }

        })
    }
}
```

We run the tests and they pass, so we're good to go.  This should give you a pretty good idea of how HTTP works under the hood.

#### Conclusion

You're rarely going to directly parse HTTP, but when things go wrong it's important to know how they actually work. The relative simplicity of the protocol should raise some eyebrows when you compare it to the incredibly overengineered complexity of the modern web. In the next article, we'll start diving in to how to deal with HTTP 'the real way' and dive into the standard library's `net/http` package.

Like this article? Need help making great software, or just want to save a couple hundred thousand dollars on your cloud bill? Hire me, or bring me in to consult. Professional enquiries at
[efron.dev@gmail.com](efron.dev@gmail.com) or [linkedin](https://www.linkedin.com/in/efronlicht)
