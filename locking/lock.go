package locking

// Lock.
//
// Represents the state of a single lock.
type lockImpl struct {
	tickets []*ticketImpl
}
