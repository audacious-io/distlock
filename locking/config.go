package locking

import (
	"time"
)

// Locking manager configuration.
type Config struct {
	// Maintenance interval.
	//
	// Defaults to 10 milliseconds.
	MaintenanceInterval time.Duration
}
