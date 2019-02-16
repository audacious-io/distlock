package locking

import (
	"testing"
	"time"
)

const timeScale = 100 * time.Millisecond

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

func TestManagerAcquireSecondTimesOutImmediately(t *testing.T) {
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
	ticketB, err := manager.Acquire("a", 0, 20*timeScale)

	if err != nil {
		t.Fatalf("Unexpected error acquiring lock: %v", err)
	}
	if ticketB == nil {
		t.Fatalf("Acquired ticket is unexpectedly nil")
	}

	select {
	case status := <-ticketB.Acquired():
		if status {
			t.Fatalf("Lock was unexpectedly acquired after waiting for timeout")
		}
	default:
		t.Fatalf("Lock did not report acquisition state after timeout")
	}

	// Assert that the path is indeed locked.
	AssertPathLocked(t, manager, "a", ticketA.Id())
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

func TestManagerAcquireSecondAcquiresAfterFirstExtendedTimeout(t *testing.T) {
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

	// Extend lock A by another period.
	found, err := manager.Extend("a", ticketA.Id(), 10*timeScale)
	if err != nil {
		t.Fatalf("Failed to extend lock: %v", err)
	}
	if !found {
		t.Fatalf("Lock was not found when trying to extend: %v", err)
	}

	// Attempt to extend lock B.
	found, err = manager.Extend("a", ticketB.Id(), 10*timeScale)
	if err != nil {
		t.Fatalf("Failed to extend lock: %v", err)
	}
	if found {
		t.Fatalf("Lock was unexpectedly found when trying to extend: %v", err)
	}

	AssertPathLocked(t, manager, "a", ticketA.Id())

	// Assert that after another second, the lock is still held by ticket A.
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

func TestManagerInspect(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	// Test inspecting with no locks held.
	state, err := manager.Inspect("a")
	if err != nil {
		t.Fatalf("Failed to inspect lock: %v", err)
	}

	if state.LockingId != 0 {
		t.Errorf("Expected no locker")
	}
	if state.LockTimeout != 0 {
		t.Errorf("Expected no lock timeout")
	}
	if len(state.Acquirers) > 0 {
		t.Errorf("Expected no acquirers")
	}

	// Test inspecting with one lock.
	ticketA, _ := manager.Acquire("a", 10*timeScale, 10*timeScale)

	state, err = manager.Inspect("a")
	if err != nil {
		t.Fatalf("Failed to inspect lock: %v", err)
	}

	if state.LockingId != ticketA.Id() {
		t.Errorf("Expected lock to be held by %d", ticketA.Id())
	}
	if state.LockTimeout <= 0 {
		t.Errorf("Expected lock timeout")
	}
	if len(state.Acquirers) > 0 {
		t.Errorf("Expected no acquirers")
	}

	// Test inspecting with multiple locks.
	ticketB, _ := manager.Acquire("a", 10*timeScale, 10*timeScale)
	ticketC, _ := manager.Acquire("a", 10*timeScale, 10*timeScale)
	manager.Acquire("b", 10*timeScale, 10*timeScale)

	state, err = manager.Inspect("a")
	if err != nil {
		t.Fatalf("Failed to inspect lock: %v", err)
	}

	if state.LockingId != ticketA.Id() {
		t.Errorf("Expected lock to be held by %d", ticketA.Id())
	}
	if state.LockTimeout <= 0 {
		t.Errorf("Expected lock timeout")
	}
	if len(state.Acquirers) != 2 {
		t.Errorf("Expected 2 acquirers")
	}

	if state.Acquirers[0].Id != ticketB.Id() {
		t.Errorf("Expected acquirer #1 to be %d", ticketB.Id())
	}
	if state.Acquirers[0].Timeout <= 0 {
		t.Errorf("Expected acquirer #1 to have timeout")
	}
	if state.Acquirers[1].Id != ticketC.Id() {
		t.Errorf("Expected acquirer #2 to be %d", ticketC.Id())
	}
	if state.Acquirers[1].Timeout <= 0 {
		t.Errorf("Expected acquirer #2 to have timeout")
	}
}

func TestManagerExtendNonExistent(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	found, err := manager.Extend("a", 1, time.Second)
	if err != nil {
		t.Fatalf("Failed to extend lock: %v", err)
	}
	if found {
		t.Fatalf("Lock unexpectedly found")
	}
}

func TestManagerReleaseNonExistent(t *testing.T) {
	manager := NewManager(Config{MaintenanceInterval: timeScale})
	go manager.Start()
	defer manager.Stop()

	found, err := manager.Release("a", 1)
	if err != nil {
		t.Fatalf("Failed to release lock: %v", err)
	}
	if found {
		t.Fatalf("Lock unexpectedly found")
	}
}

func AssertPathLocked(t *testing.T, manager Manager, path string, expected int64) {
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
