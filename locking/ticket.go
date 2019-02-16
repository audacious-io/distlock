package locking

import (
	"time"
)

// Lock ticket.
//
// If the ticket is the current lock holder, it will have its lease timeout set. If not, it will have its acquisition
// timeout set. Note that once a ticket is dereferenced by the manager, these rules no longer hold.
type Ticket interface {
	// Ticket ID.
	//
	// Identifies the specific locking attempt or lease.
	Id() uint64

	// Acquired.
	//
	// Channel that will eventually emit the state of the acquisition attempt of the ticket.
	Acquired() <-chan bool
}

// Lock ticket implementation.
type ticketImpl struct {
	// Lease ID.
	id uint64

	// First lease timeout upon acquisition.
	firstLeaseTimeout time.Duration

	// Acquisition notification channel.
	acquiredChan chan bool

	// Acquisition timeout as UNIX nanosecond timestamp.
	acquireTimeoutAt int64

	// Lease timeout as UNIX nanosecond timestamp.
	leaseTimeoutAt int64
}

func (t *ticketImpl) Id() uint64 {
	return t.id
}

func (t *ticketImpl) Acquired() <-chan bool {
	return t.acquiredChan
}
