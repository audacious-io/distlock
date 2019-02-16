package locking

import (
	"time"
)

// Llock acquirer state.
type LockAcquirerState struct {
	// ID.
	Id int64

	// Timeout.
	Timeout time.Duration
}

// Lock state.
type LockState struct {
	// Locking lease ID.
	//
	// Zero if the lock is not currently held.
	LockingId int64

	// Lock timeout.
	LockTimeout time.Duration

	// Waiting acquirers.
	Acquirers []LockAcquirerState
}
