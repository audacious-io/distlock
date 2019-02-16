package locking

import (
	"errors"
	"regexp"
)

// Valid path expression.
var validPathExpr = regexp.MustCompile(`^[\w\-]+(?:\/[\w\-]+)*$`)

// Invalid path.
var ErrPathInvalid = errors.New("invalid path")

// Validate lock path.
//
// Cleans and validates the provided lock path, returning an error if the path is not valid.
func ValidateLockPath(path string) (string, error) {
	// Strip trailing slash.
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Ensure that the path follows the following basic rule:
	//
	// 1. It does not end in a trailing slash.
	// 2. There are no empty path segments.
	if !validPathExpr.MatchString(path) {
		return path, ErrPathInvalid
	}

	return path, nil
}
