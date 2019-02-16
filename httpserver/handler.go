package httpserver

import (
	"fmt"
	"net/http"
	"strconv"

	"distlock/locking"
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

	if req.Method == "POST" {
		err = h.serveAcquire(resp, req)
	} else if req.Method == "DELETE" {
		err = h.serveRelease(resp, req)
	} else if req.Method == "PATCH" {
		err = h.serveExtend(resp, req)
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
