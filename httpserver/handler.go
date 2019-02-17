package httpserver

import (
	"fmt"
	"net/http"
	"strconv"

	"lockerd/locking"
)

// HTTP handler for the locking API.
type handler struct {
	manager locking.Manager
}

// New handler.
func NewHandler(manager locking.Manager) http.Handler {
	return &handler{
		manager: manager,
	}
}

func (h *handler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var err error

	switch req.Method {
	case "POST":
		err = h.serveAcquire(resp, req)
	case "DELETE":
		err = h.serveRelease(resp, req)
	case "PATCH":
		err = h.serveExtend(resp, req)
	case "GET":
		if req.URL.Path == "/" {
			err = h.serveInspectAll(resp, req)
		} else {
			err = h.serveInspect(resp, req)
		}
	}

	if err != nil {
		respondError(resp, "internal_server_error", "Internal server error", 500)
	}
}

func (h *handler) serveAcquire(resp http.ResponseWriter, req *http.Request) error {
	// Parse the path.
	path, err := locking.ValidateLockPath(req.URL.Path)
	if err != nil {
		return respondNotFound(resp)
	}

	// Parse the timeout values.
	lockTimeoutStr := req.FormValue("lock_timeout")
	leaseTimeoutStr := req.FormValue("lease_timeout")

	if lockTimeoutStr == "" {
		return respondError(resp, "missing_lock_timeout", "Missing form parameter lock_timeout", 400)
	}
	if leaseTimeoutStr == "" {
		return respondError(resp, "missing_lease_timeout", "Missing form parameter lease_timeout", 400)
	}

	lockTimeout, err := ParseDuration(lockTimeoutStr)
	if err != nil {
		return respondError(resp, "invalid_lock_timeout", "Invalid lock timeout", 400)
	}
	leaseTimeout, err := ParseDuration(leaseTimeoutStr)
	if err != nil {
		return respondError(resp, "invalid_lease_timeout", "Invalid lease timeout", 400)
	}

	// Acquire the lock.
	ticket, err := h.manager.Acquire(path, lockTimeout, leaseTimeout)
	if err != nil {
		return err
	}

	select {
	case acquired := <-ticket.Acquired():
		if acquired {
			return respondJson(resp, map[string]interface{}{
				"id": fmt.Sprintf("%d", ticket.Id()),
			}, 200)
		} else {
			return respondError(resp, "timeout", "Timed out waiting to acquire lock", 408)
		}

	case <-req.Context().Done():
		h.manager.Release(path, ticket.Id())
	}

	return nil
}

func (h *handler) serveRelease(resp http.ResponseWriter, req *http.Request) error {
	// Parse the path.
	path, err := locking.ValidateLockPath(req.URL.Path)
	if err != nil {
		return respondNotFound(resp)
	}

	// Parse the timeout values.
	idStr := req.FormValue("id")

	if idStr == "" {
		return respondError(resp, "missing_id", "Missing form parameter id", 400)
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return respondError(resp, "invalid_id", "Invalid ID", 400)
	}

	// Release the lock.
	released, err := h.manager.Release(path, id)
	if err != nil {
		return err
	}

	if released {
		return respondJson(resp, map[string]interface{}{}, 200)
	}

	return respondNotFound(resp)
}

func (h *handler) serveExtend(resp http.ResponseWriter, req *http.Request) error {
	// Parse the path.
	path, err := locking.ValidateLockPath(req.URL.Path)
	if err != nil {
		return respondNotFound(resp)
	}

	// Parse the timeout values.
	idStr := req.FormValue("id")
	leaseTimeoutStr := req.FormValue("lease_timeout")

	if idStr == "" {
		return respondError(resp, "missing_id", "Missing form parameter id", 400)
	}

	if leaseTimeoutStr == "" {
		return respondError(resp, "missing_lease_timeout", "Missing form parameter lease_timeout", 400)
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return respondError(resp, "invalid_id", "Invalid ID", 400)
	}
	leaseTimeout, err := ParseDuration(leaseTimeoutStr)
	if err != nil {
		return respondError(resp, "invalid_lease_timeout", "Invalid lease timeout", 400)
	}

	// Extend the lock.
	extended, err := h.manager.Extend(path, id, leaseTimeout)
	if err != nil {
		return err
	}

	if extended {
		return respondJson(resp, map[string]interface{}{}, 200)
	}

	return respondNotFound(resp)
}

func (h *handler) serveInspect(resp http.ResponseWriter, req *http.Request) error {
	// Parse the path.
	path, err := locking.ValidateLockPath(req.URL.Path)
	if err != nil {
		return respondNotFound(resp)
	}

	// Inspect the lock.
	state, err := h.manager.Inspect(path)
	if err != nil {
		return err
	}

	if state.LockingId == 0 {
		return respondNotFound(resp)
	}

	acquirers := make([]interface{}, len(state.Acquirers))
	for idx, acquirer := range state.Acquirers {
		acquirers[idx] = map[string]interface{}{
			"id":      fmt.Sprintf("%d", acquirer.Id),
			"timeout": FormatDuration(acquirer.Timeout),
		}
	}

	return respondJson(resp, map[string]interface{}{
		"locking_id":   fmt.Sprintf("%d", state.LockingId),
		"lock_timeout": FormatDuration(state.LockTimeout),
		"acquirers":    acquirers,
	}, 200)
}

func (h *handler) serveInspectAll(resp http.ResponseWriter, req *http.Request) error {
	// Inspect the manager.
	states, err := h.manager.InspectAll()
	if err != nil {
		return err
	}

	locks := make(map[string]interface{}, len(states))

	for path, state := range states {
		acquirers := make([]interface{}, len(state.Acquirers))
		for idx, acquirer := range state.Acquirers {
			acquirers[idx] = map[string]interface{}{
				"id":      fmt.Sprintf("%d", acquirer.Id),
				"timeout": FormatDuration(acquirer.Timeout),
			}
		}

		locks[path] = map[string]interface{}{
			"locking_id":   fmt.Sprintf("%d", state.LockingId),
			"lock_timeout": FormatDuration(state.LockTimeout),
			"acquirers":    acquirers,
		}
	}

	return respondJson(resp, locks, 200)
}
