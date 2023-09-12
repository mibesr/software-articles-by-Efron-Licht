package trace

import "github.com/google/uuid"

type Trace struct {
	TraceID   uuid.UUID // unique id shared by all requests and responses in a single trace; put/retrieve from 'X-Trace-Id' header
	RequestID uuid.UUID // unique id for a single request/response pair in a trace; put/retrieve from 'X-Request-Id' header
}
