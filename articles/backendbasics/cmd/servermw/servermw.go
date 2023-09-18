package servermw

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/ctxutil"
	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/trace"
)

// Default returns a middleware that combines the Recovery, RecordResponse, Log, and Trace middlewares, applying them Last-In, First-Out.
func Default(h http.Handler) http.Handler { return Recovery(RecordResponse(Log(Trace(h)))) }

// Recovery returns a middleware that recovers from panics, writing a 500 status code and "internal server error" message to the response,
// and logging the panic and associated stack trace.
func Recovery(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() { // recover from panic
			err := recover()
			if err == nil {
				return // no panic; nothing to do
			}
			// log the panic and stack trace

			if logger, ok := ctxutil.Value[*log.Logger](r.Context()); ok {
				logger.Printf("%s %s: panic: %v\n%s", r.Method, r.URL, err, debug.Stack())
			} else { // use default logger
				log.Printf("panic: %v\n%s", err, debug.Stack())
			}
			// write 500 status code and "internal server error" message to response so it doesn't hang
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("500 Internal Server Error"))
		}()
		h.ServeHTTP(w, r)
	}
}

// Trace returns a middleware that injects a trace into the request context,
// picking up the trace id from the request header if it exists, or generating a new one if it doesn't.
// This should fire BEFORE the Log middleware, if you're using it.
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
// See clientmw.Log for the client-side implementation.
func Log(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trace, ok := ctxutil.Value[trace.Trace](r.Context())
		var prefix string
		if ok {
			// like GET /articles: [trace-id request-id]:
			prefix = fmt.Sprintf("server: %s %s: [%s %s]: ", r.Method, r.URL, trace.TraceID, trace.RequestID)
		} else {
			// like GET /articles:
			prefix = fmt.Sprintf("server: %s %s: ", r.Method, r.URL)
		}
		logger := log.New(os.Stderr, prefix, log.LstdFlags)
		ctx := ctxutil.WithValue(r.Context(), logger)
		r = r.Clone(ctx)
		h.ServeHTTP(w, r)
	}
}

// RecordResponse returns a middleware that records the response status code and total bytes written to the response.
// This should fire AFTER the Log middleware, if you're using it.
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
// It is used by the RecordResponse middleware.
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
