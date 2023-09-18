// see: https://go.dev/play/p/BBGLxqepogO
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	_ "time/tzdata"

	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/ctxutil"
	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/servermw"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()

	h, err := buildBaseRouter()
	if err != nil {
		log.Fatal(err)
	}
	h = applyMiddleware(h)

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
	demo(*port)
}

// buildBaseRouter builds the base router by mapping patterns and methods to handlers.
func buildBaseRouter() (http.Handler, error) {
	// register routes.
	r := new(Router) // we'll add routes to this router.
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
			},
		},

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

// apply middleware to the router.
// remember, middleware is applied in First In, Last Out order.
func applyMiddleware(h http.Handler) http.Handler {
	h = servermw.RecordResponse(h)
	h = servermw.Recovery(h)
	h = servermw.Log(h)
	h = servermw.Trace(h)
	return h
}
