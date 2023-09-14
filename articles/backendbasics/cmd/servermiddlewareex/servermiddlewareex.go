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
