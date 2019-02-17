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

// Lock state from lock.
func lockStateFromLock(lock *lockImpl, monotimeNow time.Duration) (state LockState) {
	state.LockingId = lock.tickets[0].id
	state.LockTimeout = lock.tickets[0].leaseTimeoutAt - monotimeNow
	state.Acquirers = make([]LockAcquirerState, len(lock.tickets)-1)

	for idx, ticket := range lock.tickets[1:] {
		state.Acquirers[idx].Id = ticket.id
		state.Acquirers[idx].Timeout = ticket.acquireTimeoutAt - monotimeNow
	}

	return
}
