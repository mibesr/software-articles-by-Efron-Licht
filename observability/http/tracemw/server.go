package tracemw

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"
	"gitlab.com/efronlicht/blog/observability/trace"
	"go.uber.org/zap"
)

// HttpServerTraceMiddleware retrieves a trace from the http headers, adds a new RequestID to the chain, and adds the trace to the request's context before calling the original handler h.
// A missing or invalid trace will generate a new trace instead.
// logError is an optional parameter for when FromHttpHeader returns an error; if nil, it's a no-op.
func Server(h http.Handler, logger *zap.Logger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		t, err := trace.FromHttpHeader(r.Header)
		if err != nil {
			t.TraceID = uuid.New()
		}
		logger := logger.With(zap.String("method", r.Method), zap.String("path", r.URL.Path))
		t.RequestIDs = append(t.RequestIDs, uuid.New())
		trace.PopulateHttpHeader(w.Header(), t)
		prefix := fmt.Sprintf("server: %s %s: ", r.Method, r.URL.Path)
		{ // log request
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := r.Header.WriteSubset(buf, excludeHeaders); err != nil {
				panic(err)
			}
			logger.Debug(prefix+"begin",
				zap.String("user-agent", r.UserAgent()),
				zap.Stringer("trace_id", t.TraceID),
				zap.Stringers("request_id", t.RequestIDs),
				zap.String("remote_addr", r.RemoteAddr),
				zap.Stringer("headers", buf),
			)
			bufpool.Put(buf)
		}

		lw := &writer{ResponseWriter: w}
		defer func() {
			elapsed := time.Since(start)
			if p := recover(); p != nil {
				lw.WriteHeader(500)
				logger.Error(prefix+"end: panic", zap.Any("panic", p), zap.ByteString("stack", debug.Stack()), zap.Int("status_code", lw.statusCode), zap.Int("content_length", lw.contentLength))
				return
			}
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := r.Header.WriteSubset(buf, excludeHeaders); err != nil {
				panic(err)
			}
			if lw.statusCode >= 300 {
				logger.Error(prefix+"end: error", zap.Int("status_code", lw.statusCode), zap.Duration("elapsed", elapsed), zap.Stringer("headers", buf))
				return
			}
			logger.Info(prefix+"end: ok", zap.Int("status_code", lw.statusCode), zap.Int("content_length", lw.contentLength), zap.Duration("elapsed", elapsed), zap.Stringer("headers", buf))
		}()
		h.ServeHTTP(lw, r.WithContext(trace.SaveCtx(r.Context(), t)))
	}
}

// loggingWriter intercepts calls to WriteHeader() and Write(), recording the status code and the total number of bytes written to the response body.
type writer struct {
	http.ResponseWriter
	statusCode, contentLength int
}

func (w *writer) Write(b []byte) (int, error) {
	if w.statusCode < 200 {
		w.WriteHeader(200)
	}
	n, err := w.ResponseWriter.Write(b)
	w.contentLength += n
	return n, err
}

func (w *writer) WriteHeader(statusCode int) {
	if w.statusCode < 200 {
		w.statusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}
