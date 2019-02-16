package httpserver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// Invalid duration.
var ErrInvalidDuration = errors.New("invalid duration")

// Valid duration expression.
var durationExpr = regexp.MustCompile(`^(\d+)(ms|s|m|h)$`)

// Parse a duration.
func ParseDuration(dur string) (time.Duration, error) {
	// Handle the special case of a zero duration.
	if dur == "0" {
		return 0, nil
	}

	// Match the duration expression.
	match := durationExpr.FindStringSubmatch(dur)
	if match == nil {
		return 0, ErrInvalidDuration
	}

	// Parse the duration into something useful.
	numerator, _ := strconv.ParseInt(match[1], 10, 64)
	result := time.Duration(numerator)

	if match[2] == "ms" {
		result = result * time.Millisecond
	} else if match[2] == "s" {
		result = result * time.Second
	} else if match[2] == "m" {
		result = result * time.Minute
	} else if match[2] == "h" {
		result = result * time.Hour
	}

	return result, nil
}

// Format a duration.
func FormatDuration(dur time.Duration) string {
	if dur < 0 {
		return "0"
	}

	if dur >= 5*time.Second {
		return fmt.Sprintf("%.3fs", float64(dur)/float64(time.Second))
	}

	return fmt.Sprintf("%.3fms", float64(dur)/float64(time.Millisecond))
}
