package enve

import (
	"strconv"
	"time"

	"gitlab.com/efronlicht/enve/parse"
)

// MustParse looks up the envvar, panicking on a missing value. Unlike "Or", the empty string is OK.
func MustString(key string) string { return Must(parse.NoOp, key) }

// MustBool looks up the envvar, parsing as with parse.Bool, panicking on a missing value or error parsing.
func MustBool(key string) bool { return Must(strconv.ParseBool, key) }

// MustDuration looks up the envvar, parsing as with parse.Duration, panicking on a missing value or error parsing.
func MustDuration(key string) time.Duration { return Must(time.ParseDuration, key) }

// MustTimeRFC3339 looks up and parses the envvar as a RFC3339 datetime, panicking on a missing value or failed parse.
func MustTimeRFC3339(key string) time.Time { return Must(parse.TimeRFC3339, key) }

// MustTimeRFC3339 looks up and parses the envvar as a uint64, panicking on a missing value or failed parse.
func MustUint64(key string) uint64 { return Must(parse.Uint, key) }

// MustTimeRFC3339 looks up and parses the envvar as an int, panicking on a missing value or failed parse.
func MustInt(key string) int { return Must(strconv.Atoi, key) }

// MustTimeRFC3339 looks up and parses the envvar as a float64, panicking on a missing value or failed parse.
func MustFloat(key string) float64 { return Must(parse.Float, key) }
