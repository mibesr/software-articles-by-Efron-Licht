// 100% statement coverage of the router.
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
var (
	client *http.Client
	server *httptest.Server
)

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

func TestRouterError(t *testing.T) {
	var r Router
	if err := r.AddRoute("", nil, ""); err == nil {
		t.Errorf("AddRoute(%q, %v, %q) returned nil, want error", "", nil, "")
	}
	if err := r.AddRoute("/{a:.+}/{a:.+}", nil, ""); err == nil {
		t.Errorf("AddRoute(%q, %v, %q) returned nil, want error", "/{a:.+}/{a:.+}", nil, "")
	}
}

func TestRouter(t *testing.T) {
	var r Router
	for _, route := range []struct {
		pattern, method string
		handler         http.HandlerFunc
	}{
		{"/", "GET", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Hello, world!\r\n")
		}},
		{
			"/echo/{a:.+}/{b:.+}/{c:.+}", "GET",
			func(w http.ResponseWriter, r *http.Request) {
				vars, _ := ctxutil.Value[PathVars](r.Context())
				_ = json.NewEncoder(w).Encode(vars)
			},
		},
		{
			"/hello/{name:[a-zA-Z]+}", "GET", func(w http.ResponseWriter, r *http.Request) {
				vars, _ := ctxutil.Value[PathVars](r.Context())
				fmt.Fprintf(w, "Hello, %s!\r\n", vars["name"])
			},
		},
	} {
		if err := r.AddRoute(route.pattern, route.handler, route.method); err != nil {
			t.Fatalf("AddRoute(%q, %v, %q) returned error: %v", route.pattern, route.handler, route.method, err)
		}
	}
	for _, tt := range []struct {
		path, method, want string
	}{
		{"/", "GET", "Hello, world!\r\n"},
		{"/hello/efron", "GET", "Hello, efron!\r\n"},
		{"/hello/efron", "POST", "404 page not found\n"},
		{"/hello/efron", "PUT", "404 page not found\n"},
		{"/echo/first/second/third", "GET", `{"a":"first","b":"second","c":"third"}` + "\n"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tt.method, tt.path, nil)
		r.ServeHTTP(rec, req)
		if got := rec.Body.String(); !strings.Contains(got, tt.want) {
			t.Errorf("r.ServeHTTP(%q, %q) returned %q, want %q", tt.method, tt.path, got, tt.want)
		}

	}
}

func TestRouteVars(t *testing.T) {
	for name, tt := range map[string]struct {
		pattern string
		path    string
		want    PathVars
		wantErr bool
	}{
		"no path params, no regexp": {
			pattern: "/chess/replay",
			path:    "/chess/replay",
		},
		"regexp, no path params": {
			pattern: "/rng/seed/{[0-9]+}",
			path:    "/rng/seed/1234",
		},
		"regexp w/ path params": {
			pattern: "/chess/replay/{white:[a-zA-Z]+}/{black:[a-zA-Z]+}/{id:[0-9]+}",
			path:    "/chess/replay/efronlicht/bobross/1234",
			want:    PathVars{"white": "efronlicht", "black": "bobross", "id": "1234"},
		},
		"bad regexp": {
			pattern: "/badregexp/{[-}",
			path:    "/badregexp/1234",
			wantErr: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			re, names, err := buildRoute(tt.pattern)
			if err != nil && tt.wantErr {
				return
			} else if err != nil {
				t.Fatalf("buildRoute(%q) returned error: %v", tt.pattern, err)
			}
			vars := pathVars(re, names, tt.path)
			if tt.want == nil {
				tt.want = make(PathVars)
			}
			if !reflect.DeepEqual(vars, tt.want) {
				t.Errorf("pathVars(%q, %q) returned %v, want %v", tt.pattern, tt.path, vars, tt.want)
			}
		})
		t.Run("no match", func(t *testing.T) {
			pattern := "/chess/replay/{white:[a-zA-Z]+}/{black:[a-zA-Z]+}/{id:[0-9]+}"
			path := "/chess/replay/efronlicht/bobross/aaa"
			re, names, err := buildRoute(pattern)
			if err != nil {
				t.Fatalf("buildRoute(%q) returned error: %v", pattern, err)
			}
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("pathVars(%q, %q) did not panic, want panic", pattern, path)
				}
			}()
			_ = pathVars(re, names, path)
		})

	}
}
