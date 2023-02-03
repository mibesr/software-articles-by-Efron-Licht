package tracemw

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"

	"gitlab.com/efronlicht/blog/observability/trace"
	"go.uber.org/zap"
)

// ClientInterface's Do() does a single http request.
// Unlike a http.RoundTripper, it may modify the outgoing request.
// Repeated calls to Do() must not interfere with each other.
// The most common ClientInterface is *http.Client.
type ClientInterface interface {
	Do(r *http.Request) (*http.Response, error)
}

var excludeHeaders = map[string]bool{
	http.CanonicalHeaderKey("Authorization"): true,
}

// ClientFunc implements *http.RoundTripper and Do()
type ClientFunc func(*http.Request) (*http.Response, error)

var bufpool = sync.Pool{New: func() any { return bytes.NewBuffer(make([]byte, 0, 256)) }}

// HTTPClientMW logs and traces a request.
// It does the following:
//   - populates the request headers with a Trace before sending off a request.
//   - logs an outgoing request at Debug level.
//   - logs an incoming response at Info or Error level.
//
// client should be a *http.Client or other item implementing the Do() interface.
//
//	//Basic Usage:
//	c := Client(http.DefaultClient, zap.L())
//	req, _ := http.NewRequest("GET", "https://example.com/ping", nil)
//	resp, err := c.Do(req)
func Client(
	client ClientInterface,
	log *zap.Logger,
) ClientInterface {
	if client == nil {
		panic("nil client")
	}
	if log == nil {
		panic("nil logger")
	}
	zap.NewNop().WithOptions()
	return ClientFunc(func(req *http.Request) (*http.Response, error) {
		t := trace.FromCtxOrNew(req.Context())
		start := time.Now()
		log := log.With(zap.String("method", req.Method), zap.String("path", req.URL.Path))
		prefix := fmt.Sprintf("client: %s %s: ", req.Method, req.URL.Path)

		{ // log request
			buf := bufpool.Get().(*bytes.Buffer)
			buf.Reset()
			if err := req.Header.WriteSubset(buf, excludeHeaders); err != nil {
				panic(err)
			}
			log.Debug(prefix+"begin",
				zap.String("user-agent", req.UserAgent()),
				zap.Stringer("trace_id", t.TraceID),
				zap.Stringers("request_id", t.RequestIDs),
				zap.String("remote_addr", req.RemoteAddr),
				zap.Stringer("headers", buf),
			)
			bufpool.Put(buf)
		}

		trace.PopulateHttpHeader(req.Header, t)
		resp, err := client.Do(req)
		if err != nil {
			log.Error(prefix+"end: request failed", zap.Error(err))
			return resp, err
		}
		if returnedTrace, ok := trace.FromCtx(req.Context()); ok {
			t = returnedTrace
		} else {
			log.Debug(prefix + "response failed to return trace")
		}
		if resp.StatusCode >= 300 {
			log.Error(prefix+"end: unexpected status code", zap.Duration("elapsed", time.Since(start)), zap.Int("status_code", resp.StatusCode), zap.Stringer("trace_id", t.TraceID), zap.Stringers("request_id", t.RequestIDs))
			return resp, err
		}

		log.Debug(prefix+"end: ok", zap.Duration("elapsed", time.Since(start)), zap.Int("status_code", resp.StatusCode), zap.Stringer("trace_id", t.TraceID), zap.Stringers("request_id", t.RequestIDs))
		return resp, err
	})
}

func (cf ClientFunc) Do(req *http.Request) (*http.Response, error) {
	return cf(req)
}
