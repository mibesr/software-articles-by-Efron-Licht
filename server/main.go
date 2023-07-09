package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"gitlab.com/efronlicht/blog/observability/http/tracemw"
	"gitlab.com/efronlicht/blog/server/static"
	"gitlab.com/efronlicht/enve"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var start = time.Now()

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT)
	if err := Run(ctx); err != nil {
		cancel()
		log.Fatal(err)
	}
	cancel()
	log.Println("successful shutdown")
}

func setupLogger() *zap.Logger {
	// for larger projects, especially distributed systems, we may want to use some kind of structured logging
	// package. I like Zap and Zerolog.
	// we'll log to standard error and a gzipped file, $APPNAME_$INSTANCE_ID.log.gz
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncodeDuration = zapcore.NanosDurationEncoder
	logger := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		&zapcore.BufferedWriteSyncer{WS: os.Stderr, FlushInterval: time.Second},
		zapcore.DebugLevel,
	)) //
	zap.ReplaceGlobals(logger)
	zap.RedirectStdLog(logger)
	logger.Info("initialized logger")
	go logger.Info("metadata dump", zap.Reflect("meta", Meta))
	return logger
}

// Run the server.
func Run(ctx context.Context) (err error) {
	logger := setupLogger()
	defer logger.Sync()
	var router http.Handler // build router.
	{
		// a router just maps requests to responses.
		// we don't have complicatd requests, so we can handle the logic ourselves.
		// it's faster, too.
		router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := strings.TrimSuffix(r.URL.Path, "/")
			switch {
			case r.Method != "GET":
				w.WriteHeader(http.StatusMethodNotAllowed)
			case p == "/debug/uptime":
				elapsed := time.Since(Meta.StartTime)
				_, _ = fmt.Fprintf(w, "%3vh %02vm %02vs", math.Floor(elapsed.Hours()), math.Floor(elapsed.Minutes()), math.Floor(elapsed.Seconds()))
			case p == "/debug/meta":
				_, _ = w.Write(metaJSON)
			case p == "":
				http.Redirect(w, r, "./index.html", http.StatusPermanentRedirect)
			default:
				// fonts are immutable and large, so we can cache them for a long time.
				// everything else is tiny and might change, so we don't cache it.
				if strings.Contains(r.URL.Path, ".woff2") {
					r.Header.Add("cache-control", "immutable")
					r.Header.Add("cache-control", "max-age=604800")
					r.Header.Add("cache-control", "public")
				} else {
					r.Header.Add("cache-control", "no-cache")
				}
				static.ServeFile(w, r)
			}
		})
		// apply middleware. middleware executes Last-In, First-Out.
		router = tracemw.Server(router, logger)

	}

	server := http.Server{
		Addr:         fmt.Sprintf(":%04d", enve.IntOr("PORT", 8080)),
		Handler:      router,
		ReadTimeout:  enve.DurationOr("READ_TIMEOUT", 2*time.Second),
		WriteTimeout: enve.DurationOr("WRITE_TIMEOUT", 5*time.Second),
		IdleTimeout:  enve.DurationOr("IDLE_TIMEOUT", time.Minute),
		// don't accept new connections if already shutting down
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	logger.Sugar().Infof("took %s to start", time.Since(start))
	logger.Info("serving http", zap.String("addr", server.Addr))
	go server.ListenAndServe()
	<-ctx.Done() // wait for (ctrl+c)

	logger.Debug(fmt.Sprintf("%V: shutting down server in %s", ctx.Err(), 2*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return server.Shutdown(ctx)
}

var (
	//go:generate sh -c "git rev-parse HEAD > commit.txt
	//go:embed commit.txt
	gitCommit string
	//go:generate sh -c "git tag --points-at HEAD" > tag.txt
	//go:embed tag.txt
	gitTag string

	// Application Metadata. Everything you might want to know about the running application, all in one place.
	// This is too heavyweight to add into the logs everywhere, so we log it once at application start and just inject
	// the InstanceID into the logs. Then a search for 'metadata dump' should let you find out the rest.
	// Some may conFsider this overkill.
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
