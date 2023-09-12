package middleware

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"runtime/debug"
	"syscall"
	"time"
)

var errMissingAuthorization = errors.New("missing or improperly formed 'Authorization' header: see https://en.wikipedia.org/wiki/Basic_access_authentication")
var errBadAuthorization = errors.New("unknown or invalid username and password for basic 'Authorization' header")

// writeErr sets the Content-Type header to application/json, then writes the given error as JSON to w's body.
func writeErr(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": %q}`, err) // no need to use the JSON package for such a simple case
}

// BasicAuthMiddleware returns a middleware that checks the request's basic auth credentials using the provided checkAuth function.
func BasicAuthMiddleware(h http.Handler, checkAuth func(ctx context.Context, username, password string) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check whether the request has basic auth credentials at all
		username, password, ok := r.BasicAuth()
		if !ok { // nope; let the client know that basic auth is required
			w.Header().Add("WWW-Authenticate", `Basic realm="Restricted"`)
			writeErr(w, errMissingAuthorization, http.StatusUnauthorized)
			return
		}
		// credentials are in the right form: check whether or not they're valid.
		// how to store & check credentials is beyond the scope of this article; in our case, we'll simply hardcode a list of username-hash pairs.
		if err := checkAuth(r.Context(), username, password); err != nil {
			w.Header().Add("WWW-Authenticate", `Basic realm="Restricted"`)
			LogOrDefault(r.Context()).ErrorContext(r.Context(), "auth error", "err", err)
			writeErr(w, errBadAuthorization, http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

type key[T any] struct{}

// SaveCtx saves t in ctx, returning a new context that can be used in later handlers or middlewares to retrieve t.
// Only one value of each type can be saved in a context.
// Remember that this returns a new context, and that the original context is not modified.
func SaveCtx[T any](ctx context.Context, t T) context.Context {
	return context.WithValue(ctx, key[T]{}, t)
}

func LogOrDefault(ctx context.Context) *slog.Logger {
	if log, ok := LoadCtx[*slog.Logger](ctx); ok {
		return log
	}
	return slog.Default()
}

// LoadCtx loads t from ctx, returning t and true if t was found, and the zero value of t and false otherwise.
func LoadCtx[T any](ctx context.Context) (T, bool) {
	t, ok := ctx.Value(key[T]{}).(T)
	return t, ok
}

// MustLoadCtx loads t from ctx, panicking if t was not found.
func MustLoadCtx[T any](ctx context.Context) T { return ctx.Value(key[T]{}).(T) }

// LoggingMiddleware is a middleware that does the following
//   - inject a logger into the request's context, using SaveCtx (to be retrieved by LoadCtx in later handlers or middlewares)
//   - populate that logger with some useful context from the request (method, path, query parameters)
//   - log at debug level at the beginning of the request
//   - log at info level at the end of the request, if the status code is not 2xx, and at debug level otherwise
//   - catch & log panics at error level, writing a 500 status code to the response
func LoggingMiddleware(log *slog.Logger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// add some useful context to the logger from the request
		querykvs := make([]any, 0, (len(r.URL.Query()))*2)
		for k, v := range r.URL.Query() {
			querykvs = append(querykvs, k, v)
		}
		queryAttr := slog.Group("query", querykvs...)
		request := slog.Group("request", "method", "start", time.Now(), r.Method, "path", r.URL.Path, "query", queryAttr)
		log = log.With(request)

		// save the logger in the request's context so that it can be used by other handlers
		ctx := SaveCtx(r.Context(), log)

		start := time.Now()
		prefix := fmt.Sprintf("%s %s: ", r.Method, r.URL.Path)
		log.DebugContext(r.Context(), prefix+"begin", "query", queryAttr)

		// wrap the response writer so that we can log the status code; recordingWriter implements http.ResponseWriter
		// but keeps track of the status code written to it.
		recordingWriter := &StatusRecordingResponseWriter{RW: w}

		// after we respond, we'll log the status code and elapsed time.
		// panics will be caught and logged as well.
		defer func() {
			elapsed := time.Since(start)
			log = log.With("elapsed", elapsed, "status", recordingWriter.StatusCode)

			if p := recover(); p != nil {
				log.ErrorContext(ctx, prefix+"panic", "panic", p, "stack", string(debug.Stack()))
				recordingWriter.WriteHeader(http.StatusInternalServerError)
				return
			}
			if recordingWriter.StatusCode < 200 || recordingWriter.StatusCode >= 300 {
				log.InfoContext(ctx, prefix+"end: error")
			} else {
				log.DebugContext(ctx, prefix+"end: OK")
			}
		}()

		// call the next handler in the chain with the substituted request & response writers
		h.ServeHTTP(recordingWriter, r.WithContext(ctx))

	})
}

func addAuthHeader(r *http.Request) *http.Request { return r /*stub for demo purposes*/ }

// DoRequest is a helper function that sends the given request using the given client. It adds the following functionality:
//   - adds an authorization header to the request
//   - retries the request up to 3 times if the server is unavailable or returns a 5xx status code
//   - returns an error if the server returns a 4xx status code
//   - logs the request duration
func DoRequest(c *http.Client, r *http.Request) (*http.Response, error) {
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

// StatusRecordingResponseWriter is an http.ResponseWriter that keeps track of the status code written to it.
type StatusRecordingResponseWriter struct {
	// underlying response writer
	RW         http.ResponseWriter
	StatusCode int
}

// WriteHeader sets the status code, if it hasn't been set already.
func (w *StatusRecordingResponseWriter) WriteHeader(statusCode int) {
	if w.StatusCode == 0 {
		w.StatusCode = statusCode
	}
	w.RW.WriteHeader(statusCode)
}

// Header returns the underlying response writer's header.
func (w *StatusRecordingResponseWriter) Header() http.Header {
	return w.RW.Header()
}

// Write writes the given bytes to the underlying response writer, setting the status code to 200 if it hasn't been set already.
func (w *StatusRecordingResponseWriter) Write(b []byte) (int, error) {
	if w.StatusCode == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.RW.Write(b)
}

// for this example, both efronlicht and jdoe have the same password; "mypassword".

var creds = map[string][]byte{
	"efronlicht": sha256.New().Sum([]byte("efronlicht:mypassword")),
	"jdoe":       sha256.New().Sum([]byte("jdoe:mypassword")),
}

func hash(s string) string { return s } // TODO: use a real hash function

var errBadCredentials = errors.New("invalid username or password")

func checkPassword(username, password string) error {
	if subtle.ConstantTimeCompare(creds[username], sha256.New().Sum([]byte(username+":"+password))) != 1 {
		return errBadCredentials
	}
	return nil
}

// TimeoutMiddleware returns a middleware that sets a timeout on the request context.
func TimeoutMiddleware(h http.Handler, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
