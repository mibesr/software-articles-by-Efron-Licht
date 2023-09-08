package backendbasics

import (
	"encoding"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Header struct{ Key, Value string }

// Request is a http 1.1 request.
type Request struct {
	Method, Path, Body string
	Headers            []Header
}

var ( // assert interfaces are implemented at compile time.
	_, _ io.WriterTo              = (*Request)(nil), (*Response)(nil)
	_, _ fmt.Stringer             = (*Request)(nil), (*Response)(nil)
	_, _ encoding.TextMarshaler   = (*Request)(nil), (*Response)(nil)
	_, _ encoding.TextUnmarshaler = (*Request)(nil), (*Response)(nil)
)

// Host returns the value of the Host header, or "" if no Host header is present.
func (r *Request) Host() string {
	for _, h := range r.Headers {
		if h.Key == "Host" {
			return h.Value
		}
	}
	return ""
}

var _ io.WriterTo = &Request{}

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

	if err := printf("%s %s HTTP/1.1\r\n", r.Method, r.Path); err != nil {
		return n, err
	}

	for _, h := range r.Headers {
		if err := printf("%s: %s\r\n", h.Key, h.Value); err != nil {
			return n, err
		}

	}
	err = printf("\r\n%s\r\n", r.Body) // empty line between headers and body; empty line at end of body.
	return n, err
}

// String returns the Request as a HTTP/1.1 request string.
func (r *Request) String() string { b := new(strings.Builder); r.WriteTo(b); return b.String() }

// UnmarshalText parses the given HTTP/1.1 request string into the Request. It returns an error if the Request is invalid.
func (r *Request) UnmarshalText(text []byte) error {
	req, err := ParseRequest(string(text))
	if err != nil {
		return err
	}
	*r = req
	return nil
}

// MarshalText returns the Request as a HTTP/1.1 request string. It returns an error if the Request is invalid.
func (r Request) MarshalText() ([]byte, error) {
	if r.Method == "" {
		return nil, errors.New("empty method")
	}
	if r.Path == "" {
		return nil, errors.New("empty path")
	}
	if len(r.Headers) == 0 {
		return nil, errors.New("missing headers")
	}
	if r.Headers[0].Key != "Host" {
		return nil, errors.New("missing Host header")
	}

	return []byte(r.String()), nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var lines []string
	var i int
	for {
		j := strings.Index(s[i:], "\r\n")
		if j == -1 {
			lines = append(lines, s[i:])
			return lines
		}
		k := i + j
		lines = append(lines, s[i:k])

		i = k + 2
	}
}
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
		if key == "Host" {
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

// Response is a http 1.1 response
type Response struct {
	StatusCode int
	Body       string
	Headers    []Header
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

func (resp *Response) WithHeader(key, value string) *Response {
	resp.Headers = append(resp.Headers, Header{AsTitle(key), value})
	return resp
}
func (r *Request) WithHeader(key, value string) *Request {
	r.Headers = append(r.Headers, Header{AsTitle(key), value})
	return r
}

// newTitleCase returns the given header key as title case; e.g. "content-type" -> "Content-Type";
// it always allocates a new string.
func newTitleCase(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for i := range key {
		// the old c-style trick of using ASCII math to convert between upper and lower case can be
		// a bit confusing at first, but it's a nice trick to have in your toolbox.
		// the idea is as follows: A..=Z are 65..=90, and a..=z are 97..=122.
		// so upper-case letters are 32 less than their lower-case counterparts (or 'a'-'A' == 32).
		// rather than using the 'magic' number 32, we use 'a'-'A' to make it (somewhat) more clear what's going on.
		if i == 0 || key[i-1] == '-' {
			if key[i] >= 'a' && key[i] <= 'z' {
				b.WriteByte(key[i] + 'A' - 'a')
			} else {
				b.WriteByte(key[i])
			}
		} else if key[i] >= 'A' && key[i] <= 'Z' {
			b.WriteByte(key[i] + 'a' - 'A')
		} else {
			b.WriteByte(key[i])
		}
	}
	return b.String()
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

// AsTitle returns the given header key as title case; e.g. "content-type" -> "Content-Type"
func AsTitle(key string) string {
	if key == "" {
		panic("empty header key")
	}
	if isTitleCase(key) {
		return key
	}
	return newTitleCase(key)
}

func (resp *Response) String() string { b := new(strings.Builder); resp.WriteTo(b); return b.String() }

func (resp *Response) UnmarshalText(text []byte) error {
	r, err := ParseResponse(string(text))
	if err != nil {
		return err
	}
	*resp = *r
	return nil
}

func (resp *Response) MarshalText() ([]byte, error) {
	if resp == nil {
		return nil, errors.New("cannot marshal nil response")
	}
	if resp.StatusCode == 0 || resp.StatusCode < 0 || resp.StatusCode >= 600 {
		return nil, fmt.Errorf("invalid status code %d", resp.StatusCode)
	}
	if resp.Headers == nil {
		return nil, errors.New("nil headers")
	}
	for i, h := range resp.Headers {
		if h.Key == "" {
			return nil, fmt.Errorf("empty header key at index %d/%d", i, len(resp.Headers))
		}
		if h.Value == "" {
			return nil, fmt.Errorf("empty header value for key %q at index %d/%d", h.Key, i, len(resp.Headers))
		}
	}

	return []byte(resp.String()), nil
}

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
