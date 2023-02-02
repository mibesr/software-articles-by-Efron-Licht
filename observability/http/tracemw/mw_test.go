package tracemw_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gitlab.com/efronlicht/blog/observability/http/tracemw"
	"gitlab.com/efronlicht/blog/observability/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// built fresh by each setupAndTearDown()
var (
	buf    = new(bytes.Buffer)
	logger *zap.Logger
	client tracemw.ClientInterface
)

// call at the beginning of a test with deferSetupAndTearDown()()
// note ()()
func setupAndTearDown() func() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ping")) })
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) { http.Error(w, "500 Internal Server Error", 500) })
	logger = zap.New(zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()), zapcore.AddSync(buf), zapcore.DebugLevel))
	client = tracemw.Client(&http.Client{Timeout: 2 * time.Second}, logger)
	srv := http.Server{Addr: ":6123", Handler: tracemw.Server(mux, logger)}
	go srv.ListenAndServe()
	time.Sleep(20 * time.Millisecond)
	return srv.Close
}

func TestThreadTraceClientServerClient(t *testing.T) {
	defer setupAndTearDown()()
	reqTrc := trace.New()
	ctx := trace.SaveCtx(context.Background(), reqTrc)

	{ // GET /ping
		req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:6123/ping", nil)
		resp, _ := client.Do(req)
		respTrc, _ := trace.FromHttpHeader(resp.Header)
		if reqTrc.TraceID != respTrc.TraceID {
			t.Fatalf("expected %s, got %s", reqTrc.TraceID, respTrc.TraceID)
		}
		if reqTrc.RequestIDs[0] != respTrc.RequestIDs[0] {
			t.Fatalf("expected %s, got %s", reqTrc.TraceID, respTrc.TraceID)
		}
		if len(respTrc.RequestIDs) != 2 {
			t.Fatal("expected an additional reqID")
		}
		b, _ := io.ReadAll(resp.Body)
		if string(b) != "ping" {
			t.Fatalf("expected ping, got %s", b)
		}
		for _, s := range []string{"begin", "end", "ok"} {
			if !strings.Contains(buf.String(), "s") {
				t.Fatalf("expected logs to contain %s, but didn't", s)
			}
		}
	}
	// GET /error
	{
		req, _ := http.NewRequestWithContext(trace.SaveCtx(context.Background(), reqTrc), "GET", "http://localhost:6123/error", nil)
		req.Header.Add("foo", "bar")
		_, _ = client.Do(req)
		for _, s := range []string{"begin", "end", "error", "500"} {
			if !strings.Contains(buf.String(), "s") {
				t.Fatalf("expected logs to contain %s, but didn't", s)
			}
		}
	}

}

func TestTraceFromServerOnly(t *testing.T) {
	defer setupAndTearDown()()
	req, _ := http.NewRequestWithContext(context.Background(), "GET", "http://localhost:6123/ping", nil)

	resp, _ := http.DefaultClient.Do(req)
	trc, _ := trace.FromHttpHeader(resp.Header)
	if trc.TraceID == uuid.Nil {
		t.Fatal("no trace ID")
	}
	if trc.RequestIDs[0] == uuid.Nil {
		t.Fatal("no request ID")
	}
	for _, s := range []string{"begin", "end", "ok"} {
		if !strings.Contains(buf.String(), "s") {
			t.Fatalf("expected logs to contain %s, but didn't", s)
		}
	}

}
