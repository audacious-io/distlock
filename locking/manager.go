package locking

import (
	"time"
	"sync"
)

// Lock manager.
//
// For timeouts etc. to function properly, the maintenance of the lock manager must be started and subsequently
// stopped.
type Manager interface {
	// Start maintenance.
	Start()

	// Stop maintenance.
	Stop()

	// Release a lock.
	//
	// If the ID is for a ticket that is still waiting to be locked, the ticket is informed of failed acquisition and
	// removed from the queue. Returns whether the ticket was found.
	Release(path string, id uint64) (found bool)

	// Acquire a lock.
	//
	// Acquires a lock with a given timeout after which the attempt is aborted. The acquisition does not support
	// infinite timeouts. The lease timeout is the lifetime of the lock if not renewed after the lock is acquired.
	//
	// The function returns a ticket, that can be evaluated for the lock state. The ticket is not a guarantee, that a
	// lock can be acquired in a timely fashion. It is safe to release the ticket subsequent to acquisition no matter
	// if the ticket was actually acquired, signaling either the release of the lock or the intent to not carry on
	// with the acquisition. In the latter case, the ticket is guaranteed to indicate that acquisition failed.
	Acquire(path string, lockTimeout time.Duration, leaseTimeout time.Duration) (ticket Ticket, err error)

	// Test if a path is locked.
	//
	// Returns the ID of the ticket holding the lock if the path is locked, otherwise zero.
	IsLocked(path string) (locker uint64, err error)
}

// Lock manager implementation.
//
// Manages all available locks by path. Each individual is managed in an immutable manner, thus leading to safe
// snapshots. Note that lock tickets are not immutable, meaning that their safe state observation is subject to
// global manager locking.
//
// To avoid a large amount of updates to contended locks, maintenance, ie. the timing out of locks and waiting
// acquisitions, is, unless an explicit lock release is performed, performed in batches at a configurable interval.
//
// In its present form, the lock manager's complexity scales linearly with the number of outstanding tickets per lock
// path.
type managerImpl struct {
	sync sync.Mutex
	maintainChan chan string
	locks map[string]*lockImpl
	nextTicketId uint64
	maintenanceInterval time.Duration
}

// New lock manager.
func NewManager(config Config) Manager {
	maintenanceInterval := 10 * time.Millisecond

	if config.MaintenanceInterval > 0 {
		maintenanceInterval = config.MaintenanceInterval
	}

	return &managerImpl{
		locks: make(map[string]*lockImpl),
		nextTicketId: 1,
		maintainChan: make(chan string, 1024),
		maintenanceInterval: maintenanceInterval,
	}
}

func (m *managerImpl) Release(path string, id uint64) bool {
	// Lock the manager.
	m.sync.Lock()
	defer m.sync.Unlock()

	// Find the lock.
	curLock, ok := m.locks[path]
	if !ok || len(curLock.tickets) == 0 {
		return false
	}

	// Update the lock state.
	found := false
	nextTickets := make([]*ticketImpl, 0, len(curLock.tickets))

	for _, ticket := range curLock.tickets {
		if ticket.id == id {
			found = true

			if ticket.leaseTimeoutAt == 0 {
				// The ticket is not yet the head, so we need to emit the acquisition state.
				ticket.acquiredChan <- false
			}
		} else {
			nextTickets = append(nextTickets, ticket)
		}
	}

	// Update the lock, and, if necessary, perform maintenance.
	if len(nextTickets) > 0 {
		m.locks[path] = &lockImpl{
			tickets: nextTickets,
		}
		m.maintainPath(path)
	} else {
		delete(m.locks, path)
	}

	return found
}

// Maintain a path.
//
// This assumes exclusive lock to the manager is provided during the process.
func (m *managerImpl) maintainPath(path string) {
	curLock, ok := m.locks[path]
	if !ok || len(curLock.tickets) == 0 {
		return
	}

	// Determine which tickets are to survive.
	var nextTickets []*ticketImpl
	now := time.Now().UnixNano()

	for _, ticket := range curLock.tickets {
		if ticket.leaseTimeoutAt > 0 {
			// Locked tickets stay in place until their timeout.
			if ticket.leaseTimeoutAt > now {
				nextTickets = append(nextTickets, ticket)
			}
		} else {
			// Waiting acquisitions stay in play until their timeout.
			if ticket.acquireTimeoutAt > now {
				nextTickets = append(nextTickets, ticket)
			} else {
				ticket.acquiredChan <- false
			}
		}
	}

	// Promote the head ticket if necessary.
	if len(nextTickets) > 0 && nextTickets[0].leaseTimeoutAt == 0 {
		ticket := nextTickets[0]

		ticket.leaseTimeoutAt = time.Now().Add(ticket.firstLeaseTimeout).UnixNano()
		ticket.acquiredChan <- true

		go func() {
			<-time.After(ticket.firstLeaseTimeout)
			m.maintainChan <- path
		}()
	}

	// Update the lock state.
	if len(nextTickets) == 0 {
		delete(m.locks, path)
	} else if len(nextTickets) != len(curLock.tickets) {
		m.locks[path] = &lockImpl{
			tickets: nextTickets,
		}
	}
}

func (m *managerImpl) Start() {
	go func() {
		for {
			path := <- m.maintainChan

			m.sync.Lock()
			m.maintainPath(path)
			m.sync.Unlock()
		}
	}()
}

func (m *managerImpl) Stop() {

}

func (m *managerImpl) Acquire(path string, lockTimeout time.Duration, leaseTimeout time.Duration) (Ticket, error) {
	// Clean and validate the path.
	path, err := ValidateLockPath(path)
	if err != nil {
		return nil, err
	}

	// Lock the manager.
	m.sync.Lock()
	defer m.sync.Unlock()

	// Create a lock representation if one does not already exist for the given path.
	prevLock, _ := m.locks[path]

	// Create a ticket and evaluate locking.
	ticketId := m.nextTicketId
	m.nextTicketId++

	ticket := &ticketImpl{
		id: ticketId,
		acquiredChan: make(chan bool, 1),
		firstLeaseTimeout: leaseTimeout,
	}

	if prevLock == nil || len(prevLock.tickets) == 0 {
		// If the ticket is the new head of the lock, we set its lease timeout and informs of acquisition immediately.
		m.locks[path] = &lockImpl{
			tickets: []*ticketImpl{ticket},
		}

		ticket.leaseTimeoutAt = time.Now().Add(leaseTimeout).UnixNano()
		ticket.acquiredChan <- true

		go func() {
			<-time.After(leaseTimeout)
			m.maintainChan <- path
		}()
	} else if lockTimeout <= 0 {
		// If the lock timeout is immediate, we simply indicate that the lock could not be acquired.
		ticket.acquiredChan <- false
	} else {
		// If the ticket is not the head of the lock, we append it to the list of tickets and set its acquisition
		// timeout.
		m.locks[path] = &lockImpl{
			tickets: append(prevLock.tickets, ticket),
		}

		ticket.acquireTimeoutAt = time.Now().Add(lockTimeout).UnixNano()

		go func() {
			<-time.After(lockTimeout)
			m.maintainChan <- path
		}()
	}

	return ticket, nil
}

func (m *managerImpl) IsLocked(path string) (locker uint64, err error) {
	// Clean and validate the path.
	path, err = ValidateLockPath(path)
	if err != nil {
		return
	}

	// Lock the manager.
	m.sync.Lock()
	defer m.sync.Unlock()

	// Test the lock state.
	lock, ok := m.locks[path]
	if !ok || len(lock.tickets) == 0 {
		return
	}

	locker = lock.tickets[0].id
	return
}
