package clientmw

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/ctxutil"
	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/trace"
)

// Default returns a middleware that combines the Trace, Log, TimeRequest, and RetryOn5xx middlewares, applying them Last-In, First-Out.
// If no http.RoundTripper is provided, it will use http.DefaultTransport, just like http.Client.
func Default(h http.RoundTripper) http.RoundTripper {
	if h == nil {
		h = http.DefaultTransport
	}
	h = TimeRequest(Log(Trace(h)))
	const wait, tries = 10 * time.Millisecond, 3
	return RetryOn5xx(h, wait, tries)
}

// RoundTripFunc is an adapter to allow the use of ordinary functions as RoundTrippers, a-la http.HandlerFunc
type RoundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip implements the RoundTripper interface by calling f(r)
func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

var _ http.RoundTripper = RoundTripFunc(nil) // assert that RoundTripFunc implements http.RoundTripper at compile time

// RetryOn5xx returns a RoundTripFunc that retries the request up to n times if the server returns a 5xx status code.
// It will use exponential backoff: first retry will be after wait, second after 2*wait, third after 4*wait, etc.
func RetryOn5xx(rt http.RoundTripper, wait time.Duration, tries int) RoundTripFunc {
	// validate arguments OUTSIDE of the closure, so that it only happens once
	if tries <= 1 {
		panic("n must be > 1")
	}
	if wait <= 0 {
		panic("wait must be > 0")
	}
	return func(r *http.Request) (*http.Response, error) {
		// retry logic
		var retryErrs error
		for retry := 0; retry < tries; retry++ {
			if retry > 0 {
				time.Sleep(wait << retry)
			}
			resp, err := rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
			if errors.Is(retryErrs, syscall.ECONNREFUSED) || errors.Is(retryErrs, syscall.ECONNRESET) {
				retryErrs = errors.Join(retryErrs, err)
				continue
			}

			if err != nil {
				return nil, fmt.Errorf("failed after %d retries: %w", retry, errors.Join(retryErrs, err))
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
}

// TimeRequest returns a RoundTripFunc that logs the duration of the request.
func TimeRequest(rt http.RoundTripper) RoundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		start := time.Now()
		resp, err := rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
		if err != nil {
			log.Printf("%s %s: errored after %s", r.Method, r.URL, time.Since(start))
			return nil, err
		}
		log.Printf("%s %s: %d %s in %s", r.Method, r.URL, resp.StatusCode, http.StatusText(resp.StatusCode), time.Since(start))
		return resp, nil
	}
}

// Log wraps the given RoundTripper with a middleware that logs the request method, url, status code, and duration.
func Log(rt http.RoundTripper) RoundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		trace, ok := ctxutil.Value[trace.Trace](r.Context()) // retrieve trace from context
		var prefix string
		if ok {
			prefix = fmt.Sprintf("client: %s %s: [%s %s]: ", r.Method, r.URL, trace.TraceID, trace.RequestID)
		} else {
			prefix = fmt.Sprintf("client: %s %s: ", r.Method, r.URL)
		}

		logger := log.New(os.Stderr, prefix, log.LstdFlags|log.Lshortfile)
		ctx := ctxutil.WithValue(r.Context(), logger) // add logger to context; retrieve with ctxutil.Value[log.Logger](ctx)
		r = r.WithContext(ctx)                        // add context to request

		start := time.Now()
		resp, err := rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
		if err != nil {
			logger.Printf("errored after %s: %s", time.Since(start), err)
			return nil, err
		}
		logger.Printf("%d %s in %s", resp.StatusCode, http.StatusText(resp.StatusCode), time.Since(start))
		return resp, nil
	}
}

// Trace wraps the given RoundTripper with a middleware that injects a trace.Trace into the request context or updates a pre-existing Trace with a new RequestID.
// It will generate a new trace.TraceID if one doesn't exist, and a new trace.RequestID for each request.
func Trace(rt http.RoundTripper) RoundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		// retrieve trace from context, or create a new one if it doesn't exist
		ctx := r.Context()
		trace, ok := ctxutil.Value[trace.Trace](ctx)
		if !ok { // if trace doesn't exist, create a new one
			trace.TraceID = uuid.New()
		}
		trace.RequestID = uuid.New() // always generate a new request id for a new outgoing request

		ctx = ctxutil.WithValue(r.Context(), trace) // add trace to context; retrieve with ctxutil.Value[Trace](ctx)
		r = r.WithContext(ctx)                      // pop the context into the request

		// add trace id & request id to headers so the server can pick them up on the other end
		r.Header.Set("X-Trace-ID", trace.TraceID.String())
		r.Header.Set("X-Request-ID", trace.RequestID.String())
		return rt.RoundTrip(r) // call next middleware, or http.DefaultTransport.RoundTrip if this is the last middleware
	}
}
