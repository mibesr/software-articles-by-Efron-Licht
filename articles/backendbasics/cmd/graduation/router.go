package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/ctxutil"
)

// Router allows you to match HTTP requests to handlers based on the request path.
// It use a syntax similar to gorilla/mux:
// /path/{regexp}/{name:captured-regexp}
// AddRoute adds a route to the router.
// Vars returns the path parameters for the current request, or nil if there are none.
//
//		var r Router
//		func echoHandler(w http.ResponseWriter, r *http.Request) {
//			vars := Vars(r.Context())
//			_ = json.NewEncoder(w).Encode(vars)
//		}
//		r.AddRoute("/chess/replay/{white:[a-zA-Z]+}/{black:[a-zA-Z]+}/{id:[0-9]+}", myHandler, "GET")
//		rec := httptest.NewRecorder()
//	 ...
type Router struct {
	routes []route
}

type route struct {
	pattern *regexp.Regexp
	names   []string
	raw     string // the raw pattern string
	method  string // the HTTP method to match; if empty, all methods match.
	handler http.Handler
}

// Vars is a map of path parameters to their values. It is a unique type so that ctxutil.Value can be used to retrieve it.
type PathVars map[string]string

// Vars returns the path parameters for the current request, or nil if there are none.
func Vars(ctx context.Context) PathVars { v, _ := ctxutil.Value[PathVars](ctx); return v }

// suppose our input is /chess/replay/{white:[a-zA-Z]+}/{black:[a-zA-Z]+}/{id:[0-9]+}
// i.e, we choose the white and black players' names, and the game id.
// we'd like to match /chess/replay/efronlicht/bobross/1234
// where white=efronlicht, black=bobross, and id=1234
// we will eventually compile this into a regexp that looks like:
//
//	"^/chess/replay/([a-zA-Z]+)/([a-zA-Z]+)/([0-9]+)$"
//
// and a names slice that looks like:
//
//	[]string{"white", "black", "id"}
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

// AddRoute adds a route to the router. Method is the HTTP method to match; if empty, all methods match.
// Method will be converted to uppercase; "get", "gEt", and "GET" are all equivalent.
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

// pathVars extracts the path parameters from the path and into a map.
// --- performance design note: ---
// this is pretty inefficient, since we're re-matching the regexp.
// we could instead store the regexp and the names in the route struct, just iterate through & check for matches.
// since most paths will have very few path parameters, this will perform better and avoid extra allocs.
// additionally, we could store a small amount of storage for names directly in the route struct so as to avoid
// allocating a slice for each request.
// even better, we could make a new API for *regexp.FindStringSubmatch that _appends_ to an existing slice rather than allocating a new one,
// using a sync.Pool or something to avoid allocations entirely.
// Still, the goal here is to match gorilla/mux's API w/ as simple of an implementation as possible, so we'll leave it as-is.
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
			ctx := ctxutil.WithValue(r.Context(), vars)
			route.handler.ServeHTTP(w, r.WithContext(ctx))
			return
		}
	}
	http.NotFound(w, r) // no route matched; serve a 404
}

// ReadJSON reads a JSON object from an io.ReadCloser, closing the reader when it's done. It's primarily useful for reading JSON from *http.Request.Body.
func ReadJSON[T any](r io.ReadCloser) (T, error) {
	var v T                               // declare a variable of type T
	err := json.NewDecoder(r).Decode(&v)  // decode the JSON into v
	return v, errors.Join(err, r.Close()) // close the reader and return any errors.
}

// WriteJSON writes a JSON object to a http.ResponseWriter, setting the Content-Type header to application/json.
func WriteJSON(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(v)
}

// WriteError logs an error, then writes it as a JSON object in the form {"error": <error>}, setting the Content-Type header to application/json.
func WriteError(w http.ResponseWriter, err error, code int) {
	log.Printf("%d %v: %v", code, http.StatusText(code), err) // log the error; http.StatusText gets "Not Found" from 404, etc.
	w.Header().Set("Content-Type", "encoding/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(struct {
		Error string `json:"error"`
	}{Error: err.Error()})
}
