package locking

import (
	"testing"
)

func TestValidateLockPath(t *testing.T) {
	// Test invalid paths.
	for _, path := range []string{
		"",
		"/",
		"a/",
		"a/b/c/",
		"aø",
		"aø/b",
	} {
		_, err := ValidateLockPath(path)
		if err != ErrPathInvalid {
			t.Errorf("Expected %s to result in ErrPathInvalid, got %v", path, err)
		}
	}

	// Test valid paths.
	for path, expectedPath := range map[string]string{
		"a": "a",
		"//a": "a",
		"a-b": "a-b",
		"a-b-c/095": "a-b-c/095",
	} {
		actualPath, err := ValidateLockPath(path)
		if err != nil {
			t.Errorf("Expected %s to be a valid path", path)
		} else if actualPath != expectedPath {
			t.Errorf("Expected %s to be cleaned to %s, but it was cleaned to %s", path, expectedPath, actualPath)
		}
	}
}
