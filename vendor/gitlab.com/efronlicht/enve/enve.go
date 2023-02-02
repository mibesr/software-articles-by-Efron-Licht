package enve

import (
	"fmt"
	"os"
)

type MissingKeyError string

func (err MissingKeyError) Error() string {
	return fmt.Sprintf("missing required envvar %s", string(err))
}

type Parser[T any] func(string) (T, error)

// Lookup and parse the specified environment variable key, returning ErrorMissingKey if it's missing, and the result of parse(os.Getenv(key)) otherwise.
func Lookup[T any](parse func(s string) (T, error), key string) (T, error) {
	s, ok := os.LookupEnv(key)
	if !ok {
		return *new(T), MissingKeyError(key)
	}
	return parse(s)

}

// Or looks up & parses the specified environment variable key, or returns backup if it's missing or invalid.
func Or[T any](parse func(string) (T, error), key string, backup T) T {
	t, err := Lookup(parse, key)
	if err != nil {
		logOr(key, err, parse, backup)
		return backup
	}
	return t
}

// Must looks up & parses the specified environment variable key, panicking on an error.
func Must[T any](parse func(string) (T, error), key string) T {
	t, err := Lookup(parse, key)
	if err != nil {
		logMust(key, err, parse)
		panic(err) // reachable only if enve_logdisabled

	}
	return t
}
