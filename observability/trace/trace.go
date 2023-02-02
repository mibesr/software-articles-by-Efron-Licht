// package trace contains low-level primitives for sending traces across function and http boundaries.
// In general, this package should not be used directly; for HTTP, use the middleware in ../httpmw
// Basic usage:
// 	// CLIENT
//	r, _ := http.NewRequest("GET", "https://myapp/route", nil)
// 	PopulateRequestHeaders(r.Header)
//	SERVER
//

package trace

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// New makes a new Trace with a freshly-generated TraceID and RequestID.
func New() Trace {
	return Trace{TraceID: uuid.New(), RequestIDs: []uuid.UUID{uuid.New()}}
}

// Trace contains a TraceID and one or more RequestIDs. RequestIDs are always preserved in order of creation, oldest first.
type Trace struct {
	TraceID    uuid.UUID   `json:"trace_id,omitempty"`
	RequestIDs []uuid.UUID `json:"request_ids,omitempty"`
}

const TraceIDHeader = "E-Trace-Id"
const ReqIDHeader = "E-Req-Id"

// PopulateRequestHeaders adds the traceID and RequestIDs to the request headers.
// In general, this function should not be used directly: use the HTTPClientWrapper instead.
func PopulateHttpHeader(h http.Header, t Trace) {
	reqIDs := make([]string, len(t.RequestIDs))
	for i := range reqIDs {
		reqIDs[i] = hex.EncodeToString(t.RequestIDs[i][:])
	}
	h.Set(TraceIDHeader, t.TraceID.String())
	h[ReqIDHeader] = reqIDs
}

// ErrNoTraceIDHeader is returns FromHTTPHeader when no  "E-Trace-Id" header is ound.
var ErrNoTraceIDHeader = errors.New("no E-Trace-Id header")

// ErrNoReqIDHeader is returns FromHTTPHeader when no "E-Req-Id" header is found
var ErrNoReqIDHeader = errors.New("no E-Req-ID header")

// FromHttpReq decodes a Trace from the request's headers. In eneral, this function should not be used directly: use the ServerMiddleware instead.
func FromHttpHeader(h http.Header) (Trace, error) {
	var rawTrace = h.Get(TraceIDHeader)
	traceID, err := uuid.Parse(rawTrace)
	if err != nil {
		return Trace{}, fmt.Errorf("E-Trace-Id header had invalid value %q expected a UUID: %w", rawTrace, err)
	}

	var rawReqIds []string = h[ReqIDHeader]
	if len(rawReqIds) == 0 {
		return Trace{TraceID: traceID}, ErrNoReqIDHeader
	}
	reqIDs := make([]uuid.UUID, len(rawReqIds))
	for i := range reqIDs {
		reqIDs[i], err = uuid.Parse(rawReqIds[i])
		if err != nil {
			return Trace{TraceID: traceID}, fmt.Errorf("E-Req-Id header had invalid value at position %d: %q: expected a UUID: %w", i, rawReqIds, err)
		}
	}
	return Trace{TraceID: traceID, RequestIDs: reqIDs}, nil
}

type ctxKey struct{}

// FromCtx retrieves a trace saved with SaveCtx, returning false if none was found. Most of the time, you want FromCtxOrNew.
// For debugging, try MustFromCtx.
func FromCtx(ctx context.Context) (Trace, bool) {
	t, ok := ctx.Value(ctxKey{}).(Trace)
	if !ok {
		return t, false
	}
	if t.TraceID == (uuid.UUID{}) {
		t.TraceID = uuid.New()
	}
	if len(t.RequestIDs) == 0 {
		t.RequestIDs = []uuid.UUID{uuid.New()}
	}
	return t, true
}

// MustFromCtx is as FromCtx, but panics on a missing trace.
func MustFromCtx(ctx context.Context) Trace { return ctx.Value(ctxKey{}).(Trace) }

// FromCtxOrNew retrieves a trace from the context, creating a new one if none was found.
func FromCtxOrNew(ctx context.Context) Trace {
	t, _ := ctx.Value(ctxKey{}).(Trace)
	if t.TraceID == (uuid.UUID{}) {
		t.TraceID = uuid.New()
	}
	if len(t.RequestIDs) == 0 {
		t.RequestIDs = []uuid.UUID{uuid.New()}
	}
	return t
}

// SaveCtx returns a new context with the trace, for retrieval with FromCtx.
func SaveCtx(ctx context.Context, t Trace) context.Context {
	return context.WithValue(ctx, ctxKey{}, t)
}
