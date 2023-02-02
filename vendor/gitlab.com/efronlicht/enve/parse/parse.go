package parse

import (
	"strconv"
	"time"
)

func Uint(s string) (uint64, error)   { return strconv.ParseUint(s, 0, 64) }
func Float(s string) (float64, error) { return strconv.ParseFloat(s, 64) }
func NoOp(s string) (string, error)   { return s, nil }
func TimeRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
