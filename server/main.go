package main

//go:generate sh -c "git rev-parse HEAD > commit.txt
//go:generate sh -c "git tag --points-at HEAD" > tag.txt
import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gitlab.com/efronlicht/blog/observability/http/tracemw"
	"gitlab.com/efronlicht/blog/server/middleware"
	"gitlab.com/efronlicht/blog/server/static"
	"gitlab.com/efronlicht/enve"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	if err := Run(); err != nil {
		log.Fatal(err)
	}
}

func buildIndex(fs embed.FS) []byte {
	html := []byte(`
	<!DOCTYPE html><html><head>
	<title>index.html</title>
	<meta charset="utf-8"/>
	<link rel="stylesheet" type="text/css" href="/s.css"/>
	</head>
	<body>
	<h1> articles </h1>
`)

	for _, e := range must(fs.ReadDir(".")) {
		if n := e.Name(); strings.Contains(filepath.Ext(n), "html") {
			html = fmt.Appendf(html, `<h4><a href="/%s">%s</a>`+"\n</h4>", n, n)
		}
	}
	html = append(html, "</body>"...)
	return html
}

func setupLogger() *zap.Logger {
	// for larger projects, especially distributed systems, we may want to use some kind of structured logging
	// package. I like Zap and Zerolog.
	// we'll log to standard error and a gzipped file, $APPNAME_$INSTANCE_ID.log.gz

	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncodeDuration = zapcore.MillisDurationEncoder
	return zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		&zapcore.BufferedWriteSyncer{WS: os.Stderr, FlushInterval: time.Second},
		zapcore.DebugLevel,
	))
}

func Run() (err error) {
	// initialize logger.

	logger := setupLogger()
	logger.Info("initialized logger")
	logger.Info("dumping metadata", zap.Reflect("meta", Meta))
	defer logger.Sync()

	index := buildIndex(static.FS)

	// build a router. at the most basic level, a router just maps requests to responses.
	// for something this basic, this is about as fast as you can get.
	var router http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimSuffix(r.URL.Path, "/")
		r.Header.Add("cache-control", "no-cache")
		switch {
		case r.Method != "GET":
			w.WriteHeader(http.StatusMethodNotAllowed)
		case p == "" || p == "/" || p == "/index.html":
			w.Write(index)
		case p == "/debug/uptime":
			elapsed := time.Since(Meta.StartTime)
			_, _ = fmt.Fprintf(w, "%3vh %02vm %02vs", math.Floor(elapsed.Hours()), math.Floor(elapsed.Minutes()), math.Floor(elapsed.Seconds()))
		case p == "/debug/meta":
			_, _ = w.Write(metaJSON)
		default:
			if strings.Contains(r.URL.Path, ".woff2") {
				r.Header.Add("cache-control", "immutable")
				r.Header.Add("cache-control", "max-age=604800")
				r.Header.Add("cache-control", "public")
			}
			if path.Ext(r.URL.Path) == "" {
				r.URL.Path += ".html"
			}
			static.Server.ServeHTTP(w, r)
		}
	})

	{ // apply middleware. middleware executes Last-In, First-Out.

		router = middleware.WriteGzip(router)
		router = tracemw.Server(router, logger)

	}
	// build
	server := http.Server{
		Addr:         fmt.Sprintf(":%04d", enve.IntOr("PORT", 8080)),
		Handler:      router,
		ReadTimeout:  enve.DurationOr("READ_TIMEOUT", 2*time.Second),
		WriteTimeout: enve.DurationOr("WRITE_TIMEOUT", 5*time.Second),
		IdleTimeout:  enve.DurationOr("IDLE_TIMEOUT", time.Minute),
	}
	log.Printf("serving http at %s", server.Addr)
	go server.ListenAndServe()

	// signal handler for graceful shutdown.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT) // SIGINT is sent when you hit ctrl+c in the terminal
	// we want to intercept SIGINT so that we can flush our gzipped logs before
	logger.Info("received shutdown signal: " + (<-ch).String() + ": shutting down within 2s")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

var (

	//go:embed commit.txt
	gitCommit string
	//go:embed tag.txt
	gitTag string

	// Application Metadata. Everything you might want to know about the running application, all in one place.
	// This is too heavyweight to add into the logs everywhere, so we log it once at application start and just inject
	// the InstanceID into the logs. Then a search for 'metadata dump' should let you find out the rest.
	// This may seem like overkill, but I've wished I'd logged each and every one of these fields at one time or another.
	Meta = struct {
		AppName    string
		InstanceID string // unique for each instance of the server
		StartTime  time.Time
		Git        struct{ Tag, Commit string }
		OS         struct {
			Host string
			PID  int
			User *user.User
		}
		Runtime struct{ GOARCH, GOOS, Version string }
	}{
		InstanceID: uuid.New().String(),
		AppName:    "efronlicht/blog/server",
		Git:        struct{ Tag, Commit string }{Tag: strings.TrimSpace(gitTag), Commit: strings.TrimSpace(gitCommit)},
		OS: struct {
			Host string
			PID  int
			User *user.User
		}{
			Host: must(os.Hostname()),
			PID:  os.Getpid(),
			User: must(user.Current()),
		},
		StartTime: time.Now(),
		Runtime: struct{ GOARCH, GOOS, Version string }{
			GOARCH:  runtime.GOARCH,
			GOOS:    runtime.GOOS,
			Version: runtime.Version(),
		},
	}
	// we'll serve this when we get calls to /meta
	metaJSON = must(json.Marshal(Meta))
)

func must[T any](t T, err error) T {
	if err != nil {
		pc, file, line, _ := runtime.Caller(1)
		log.Panicf("%s: %s %04d: fatal err: %v", runtime.FuncForPC(pc).Name(), file, line, err)
	}
	return t
}
