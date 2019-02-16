package locking

import (
	"testing"
	"time"
)

const timeScale = 10 * time.Millisecond

func TestManagerAcquireInvalidPath(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	ticket, err := manager.Acquire("/", 10*timeScale, 10*timeScale)

	if ticket != nil {
		t.Errorf("Expected ticket to be nil")
	}
	if err != ErrPathInvalid {
		t.Errorf("Expected ErrPathInvalid, but got %v", err)
	}
}

func TestManagerAcquireExpires(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	ticketA, err := manager.Acquire("a", 10*timeScale, 10*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketA == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is immediately acquired.
	select {
	case status := <-ticketA.Acquired():
		if status != true {
			t.Fatalf("Lock was not immediately acquired")
		}
	default:
		t.Fatalf("No lock indication was emitted from the ticket")
	}

	// Assert that the path is indeed locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Wait for the lock to time out.
	time.Sleep(11 * timeScale)

	// Assert that the path is no longer locked.
	AssertPathLocked(t, manager, "a", 0)
}

func TestManagerAcquireSecondTimesOutWhileAcquiring(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	ticketA, err := manager.Acquire("a", 10*timeScale, 20*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketA == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is immediately acquired.
	select {
	case status := <-ticketA.Acquired():
		if status != true {
			t.Fatalf("Lock was not immediately acquired")
		}
	default:
		t.Fatalf("No lock indication was emitted from the ticket")
	}

	// Attempt to acquire the lock while another caller is holding the lock.
	ticketB, err := manager.Acquire("a", 10*timeScale, 20*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketB == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is not immediately acquired.
	select {
	case <-ticketB.Acquired():
		t.Fatalf("Lock status was unexpectedly returned immediately")
	default:
	}

	// Assert that the path is indeed locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Assert that after a second, the lock acquisition fails.
	time.Sleep(11 * timeScale)

	select {
	case status := <-ticketB.Acquired():
		if status {
			t.Fatalf("Lock was unexpectedly acquired after waiting for timeout")
		}
	default:
		t.Fatalf("Lock did not report acquisition state after timeout")
	}

	// Assert that the path is still locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Assert that after another second, the lock is released.
	time.Sleep(11 * timeScale)

	AssertPathLocked(t, manager, "a", 0)
}

func TestManagerAcquireSecondAcquiresAfterFirstTimeout(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	ticketA, err := manager.Acquire("a", 10*timeScale, 10*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketA == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is immediately acquired.
	select {
	case status := <-ticketA.Acquired():
		if status != true {
			t.Fatalf("Lock was not immediately acquired")
		}
	default:
		t.Fatalf("No lock indication was emitted from the ticket")
	}

	// Attempt to acquire the lock while another caller is holding the lock.
	ticketB, err := manager.Acquire("a", 20*timeScale, 10*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketB == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is not immediately acquired.
	select {
	case <-ticketB.Acquired():
		t.Fatalf("Lock status was unexpectedly returned immediately")
	default:
	}

	// Assert that the path is indeed locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Assert that after a half a second, the lock is still held by ticket A.
	time.Sleep(5 * timeScale)

	select {
	case <-ticketB.Acquired():
		t.Fatalf("Lock status was returned")
	default:
	}

	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Assert that after another second, the lock is now held by ticket B.
	time.Sleep(6 * timeScale)

	select {
	case status := <-ticketB.Acquired():
		if !status {
			t.Fatalf("Lock was expected to be acquired after waiting for timeout")
		}
	default:
		t.Fatalf("Lock did not report acquisition state after timeout")
	}

	AssertPathLocked(t, manager, "a", ticketB.Id())

	// Assert that after another second, the lock is released.
	time.Sleep(11 * timeScale)

	AssertPathLocked(t, manager, "a", 0)
}

func TestManagerAcquireStaggered(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	ticketA, _ := manager.Acquire("a", 40*timeScale, 10*timeScale)
	ticketB, _ := manager.Acquire("a", 40*timeScale, 10*timeScale)
	ticketC, _ := manager.Acquire("a", 40*timeScale, 10*timeScale)
	ticketD, _ := manager.Acquire("a", 40*timeScale, 10*timeScale)

	// Assert each step of the way.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	time.Sleep(11 * timeScale)
	AssertPathLocked(t, manager, "a", ticketB.Id())

	time.Sleep(10 * timeScale)
	AssertPathLocked(t, manager, "a", ticketC.Id())

	time.Sleep(10 * timeScale)
	AssertPathLocked(t, manager, "a", ticketD.Id())

	time.Sleep(10 * timeScale)
	AssertPathLocked(t, manager, "a", 0)
}

func TestManagerAcquireSecondCancelsAcquiring(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	ticketA, err := manager.Acquire("a", 10*timeScale, 20*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketA == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is immediately acquired.
	select {
	case status := <-ticketA.Acquired():
		if status != true {
			t.Fatalf("Lock was not immediately acquired")
		}
	default:
		t.Fatalf("No lock indication was emitted from the ticket")
	}

	// Attempt to acquire the lock while another caller is holding the lock.
	ticketB, err := manager.Acquire("a", 50*timeScale, 20*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketB == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is not immediately acquired.
	select {
	case <-ticketB.Acquired():
		t.Fatalf("Lock status was unexpectedly returned immediately")
	default:
	}

	// Assert that the path is indeed locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Assert that after a second, the lock is still not acquired.
	time.Sleep(11 * timeScale)

	select {
	case <-ticketB.Acquired():
		t.Fatalf("Lock status was unexpectedly returned")
	default:
	}

	// Cancel the lock ticket.
	manager.Release("a", ticketB.Id())

	select {
	case status := <-ticketB.Acquired():
		if status {
			t.Fatalf("Lock was unexpectedly acquired after waiting for timeout")
		}
	default:
		t.Fatalf("Lock did not report acquisition state after cancelation")
	}

	// Assert that the path is still locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Assert that after another second, the lock is released.
	time.Sleep(11 * timeScale)

	AssertPathLocked(t, manager, "a", 0)
}

func TestManagerAcquireCanceledImmediately(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	ticketA, err := manager.Acquire("a", 10*timeScale, 20*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketA == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is immediately acquired.
	select {
	case status := <-ticketA.Acquired():
		if status != true {
			t.Fatalf("Lock was not immediately acquired")
		}
	default:
		t.Fatalf("No lock indication was emitted from the ticket")
	}

	// Assert that the path is locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Cancel the ticket.
	manager.Release("a", ticketA.Id())

	// Assert that the path is no longer locked.
	AssertPathLocked(t, manager, "a", 0)

	// Attempt to lock the path again.
	ticketB, err := manager.Acquire("a", 10*timeScale, 20*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketB == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	// Assert that the lock is immediately acquired.
	select {
	case status := <-ticketB.Acquired():
		if status != true {
			t.Fatalf("Lock was not immediately acquired")
		}
	default:
		t.Fatalf("No lock indication was emitted from the ticket")
	}

	// Assert that the path is locked.
	AssertPathLocked(t, manager, "a", ticketB.Id())
}

func AssertPathLocked(t *testing.T, manager Manager, path string, expected uint64) {
	locker, err := manager.IsLocked("a")
	if err != nil {
		t.Fatalf("Unexpected error checking lock state for %s: %v", path, err)
	}

	if locker != expected {
		if expected != 0 {
			t.Fatalf("Expected path %s to be locked by %d", path, expected)
		} else {
			t.Fatalf("Expected path %s to be locked by %d but it is locked by %d", path, expected, locker)
		}
	}
}
