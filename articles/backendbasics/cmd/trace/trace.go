package trace

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/efronlicht/blog/articles/backendbasics/cmd/ctxutil"
)

// Trace holds the trace and request ids for a request/response pair; it can be used to trace a request through
// multiple services by passing the same trace id through all requests and responses.
type Trace struct {
	// unique id shared by all requests and responses in a single trace; put/retrieve from 'X-Trace-Id' header
	TraceID uuid.UUID `json:"trace_id,omitempty"`
	// unique id for a single request/response pair in a trace; put/retrieve from 'X-Request-Id' header
	RequestID uuid.UUID `json:"request_id,omitempty"`
}

func (t Trace) SaveToHeader(h http.Header) {
	h.Set("X-Trace-Id", t.TraceID.String())
	h.Add("X-Request-Id", t.RequestID.String())
}

func Init(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	t, ok := ctxutil.Value[Trace](ctx)
	if !ok {
		t = Trace{
			TraceID:   uuid.New(),
			RequestID: uuid.New(),
		}
	} else {
		t.RequestID = uuid.New()
	}
	return ctxutil.WithValue(ctx, t)
}

func FromHeader(h http.Header) Trace {
	traceID, err := uuid.Parse(h.Get("X-Trace-Id"))
	if err != nil {
		traceID = uuid.New()
	}
	reqID, err := uuid.Parse(h.Get("X-Request-Id"))
	if err != nil {
		reqID = uuid.New()
	}
	return Trace{TraceID: traceID, RequestID: reqID}
}
