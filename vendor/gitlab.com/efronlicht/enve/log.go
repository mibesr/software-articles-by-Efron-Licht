package enve

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	seen     = make(map[string]bool)
	mux      sync.RWMutex
	once     sync.Once
	disabled bool
)

func shouldLog(key string) bool {
	once.Do(func() { // once ever
		disabled, _ = strconv.ParseBool(os.Getenv("ENVE_LOGDISABLED"))
		if disabled {
			return //
		}
		log.Printf("enve: logging enabled. disable enve logging with environment variable ENVE_LOGDISABLED=1")
	})
	if disabled {
		return false
	}
	// check once: has anyone else logged this?
	mux.RLock()
	ok := seen[key]
	mux.RUnlock()
	if ok { //  they have.
		return false
	}
	// obtain the write lock.
	mux.Lock()
	defer mux.Unlock()
	// someone else might have got the write lock between when we released the read lock and obtained the write lock.
	if seen[key] { // they have; we have nothing to do.
		return false
	}
	// we're the first to obtain the write lock.
	seen[key] = true
	return true
}

type metadata struct {
	name, file string
	line       int
}

func (m metadata) String() string {
	return fmt.Sprintf("%s (%s %d)", m.name, m.file, m.line)
}

func callerMetadata(extraSkip int) metadata {
	pc, callerFile, callerLine, _ := runtime.Caller(skip + extraSkip)
	return metadata{
		name: trim(runtime.FuncForPC(pc).Name()),
		file: trim(callerFile),
		line: callerLine,
	}
}

// trims a function path containing "efronlicht/enve", making it start with "enve".
// EG, "users/efron/go/src/gitlab.com/efronlicht/estd/parse/IPv4" => "estd/parse.IPv4".
func trim(s string) string {
	if _, after, ok := strings.Cut(s, "efronlicht/"); ok && strings.HasPrefix(after, "estd") {
		return after
	}
	return s
}

func parserMetadata[T any](f func(string) (T, error)) metadata {
	meta := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
	parserFile, parserLine := meta.FileLine(meta.Entry())
	return metadata{
		name: trim(meta.Name()),
		file: trim(parserFile),
		line: parserLine,
	}
}

// how far up the stack to go to get caller information.
// supose we call env.DurationOr("CLIENT_TIMEOUT", 5*time.Second) from main.main().
// stack should look like this:
//
//	 callerMeta() // 0
//		log.Or[time.Duration]("CLIENT_TIMEOUT", ErrMissingKey("CLIENT_TIMEOUT", parse.Duration, 5*time.Second)) // 1
//		env.or[time.Duration](parse.Duration, "CLIENT_TIMEOUT", 5*time.Second) // 2
//		env.DurationOr("CLIENT_TIMEOUT", ErrMissingKey("CLIENT_TIMEOUT")) // 3
//		main.main() // 4 <<- this is what we want
const skip = 4

func typeOf[T any]() reflect.Type { return reflect.TypeOf((*T)(nil)).Elem() }

// Must is the default log hook for the 'Lookup' group of functions, using the log package in the stdlib. See the package README for details on
// other logging options (disabling, zap, zerolog, etc).
func logMust[T any](key string, err error, parser func(s string) (t T, err error)) {
	if !shouldLog(key) { // at most once per key
		return
	}
	// no need to tell people about the parser: that's not the problem.
	if _, ok := err.(MissingKeyError); ok {
		log.Printf("enve: FATAL ERR: missing required envvar %s; caller %v", key, callerMetadata(0))
		return
	}
	log.Printf("enve: FATAL ERR: parsing required envvar %s into type %s: %v; caller %s; parser %s:", key, typeOf[T](), err, callerMetadata(0), parserMetadata(parser))
}
func logOr[T any](key string, err error, parser func(s string) (T, error), backup T) {
	if !shouldLog(key) { // at most once per key
		return
	}
	// no need to tell people about the parser: that's not the problem.
	if _, ok := err.(MissingKeyError); ok {
		log.Printf("enve: missing optional envvar %s: falling back to default: %v; caller %v", key, backup, callerMetadata(0))
		return
	}
	log.Printf("enve: invalid optional envvar %s: %v: falling back to default  %v; caller %s; parser %s", key, err, backup, callerMetadata(0), parserMetadata(parser))
}
