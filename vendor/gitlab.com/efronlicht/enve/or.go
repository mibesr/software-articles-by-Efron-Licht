package enve

import (
	"encoding"
	"errors"
	"strconv"
	"time"

	"gitlab.com/efronlicht/enve/parse"
)

// BoolOr returns the boolean value represented by the environment variable.
// It accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False. Any other value will return backup.
func BoolOr(key string, backup bool) bool { return Or(strconv.ParseBool, key, backup) }

func notEmpty(s string) (string, error) {
	if s == "" {
		return "", errors.New("unexpected empty string")
	}
	return s, nil
}

// StringOr returns os.Getenv(key) if it's not the empty string, Or backup otherwise.
func StringOr(key, backup string) string {
	return Or(notEmpty, key, backup)
}

// GetFromText  looks up and parses a T represented by the environment variable via the type's UnmarshalText() method.
// A missing key or an error will return backup.
func FromTextOr[T any, PT interface {
	*T
	encoding.TextUnmarshaler
}](key string, backup T,
) T {
	return Or(parseFromText[T, PT], key, backup)
}
func parseFromText[T any, PT interface {
	*T
	encoding.TextUnmarshaler
}](s string) (T, error) {
	var t = PT(new(T))
	err := t.UnmarshalText([]byte(s))
	return *t, err
}

// GetFromText looks up and parses the envvar via time.ParseDuration.
// A missing key or an error will return backup.
func DurationOr(key string, backup time.Duration) time.Duration {
	return Or(time.ParseDuration, key, backup)
}

// TimeRFC3339Or looks up and parses the envvar as a datetime in RFC3339.
// A missing key or an error will return backup.
func TimeRFC3339Or(key string, backup time.Time) time.Time {
	return Or(parse.TimeRFC3339, key, backup)
}

// IntOr looks up and parses the envvar as an int.
// A missing key or an error will return backup.
func IntOr(key string, backup int) int {
	return Or(strconv.Atoi, key, backup)
}

// Uint64 looks up and parses the envvar as a uint64.
// A missing key or an error will return backup.
func Uint64(key string, backup uint64) uint64 {
	return Or(parse.Uint, key, backup)
}

// FloatOr looks up and parses the envvar as a float64.
// A missing key or an error will return backup.
func FloatOr(key string, backup float64) float64 {
	return Or(parse.Float, key, backup)
}
