package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"
)

// an example of how to use signal.NotifyContext to gracefully shutdown a server.
func main() {
	// When the OS sends us an interrupt signal (i.e, via Ctrl+C),
	// sigCtx will be cancelled.
	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	server := &http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		}),
		// BaseContext is the default context for incoming requests.
		// Since we are using a signal.NotifyContext, new connections will have an automatically-cancelled context
		// after we receive an interrupt signal.
		BaseContext: func(_ net.Listener) context.Context { return sigCtx },
		// any server, no matter how trivial, should have timeouts.
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		IdleTimeout:  time.Second,
	}
	log.Println("Starting server on :8080")
	go func() {
		defer cancel()
		server.ListenAndServe()
	}()
	<-sigCtx.Done()
	// we've received an interrupt signal.
	// let's give the server 350ms to finish off it's current connections.
	ctx, cancel := context.WithTimeout(context.Background(), 350*time.Millisecond)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
