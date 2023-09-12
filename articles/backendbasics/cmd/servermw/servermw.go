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
