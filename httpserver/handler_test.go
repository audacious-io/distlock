package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"distlock/locking"
)

type SuccessResponseAcquirer struct {
	Id      string `json:"id"`
	Timeout string `json:"timeout"`
}

type SuccessResponse struct {
	Id          string                    `json:"id"`
	LockingId   string                    `json:"locking_id"`
	LockTimeout string                    `json:"lock_timeout"`
	Acquirers   []SuccessResponseAcquirer `json:"acquirers"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func TestHandlerAcquireInvalid(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Test acquiring with missing timeouts.
	resp := Post(t, server, "/test", url.Values{
		"lock_timeout": []string{"1m"},
	})
	AssertErrorResponse(t, resp, "missing_lease_timeout", 400)

	resp = Post(t, server, "/test", url.Values{
		"lease_timeout": []string{"1m"},
	})
	AssertErrorResponse(t, resp, "missing_lock_timeout", 400)

	// Test acquiring with invalid timeouts.
	resp = Post(t, server, "/test", url.Values{
		"lease_timeout": []string{"1m"},
		"lock_timeout":  []string{"1"},
	})
	AssertErrorResponse(t, resp, "invalid_lock_timeout", 400)

	resp = Post(t, server, "/test", url.Values{
		"lease_timeout": []string{"1"},
		"lock_timeout":  []string{"1m"},
	})
	AssertErrorResponse(t, resp, "invalid_lease_timeout", 400)

	// Test acquiring with an invalid path.
	resp = Post(t, server, "/test/", url.Values{
		"lock_timeout":  []string{"1m"},
		"lease_timeout": []string{"1m"},
	})
	AssertErrorResponse(t, resp, "not_found", 404)
}

func TestHandlerAcquireSuccessful(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Test acquiring successfully.
	resp := Post(t, server, "/test", url.Values{
		"lock_timeout":  []string{"1m"},
		"lease_timeout": []string{"1m"},
	})
	body := AssertSuccessResponse(t, resp)

	if body.Id == "" {
		t.Fatalf("Expected to have received an ID")
	}
	id, _ := strconv.ParseInt(body.Id, 10, 64)

	locker, _ := manager.IsLocked("test")
	if locker != id {
		t.Fatalf("Expected requestor to be locker")
	}
}

func TestHandlerAcquireTimeout(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Acquire up front to cause waiting.
	manager.Acquire("test", time.Minute, time.Minute)

	// Test acquiring causing timeout.
	resp := Post(t, server, "/test", url.Values{
		"lock_timeout":  []string{"0"},
		"lease_timeout": []string{"100ms"},
	})
	AssertErrorResponse(t, resp, "timeout", 408)
}

func TestHandlerReleaseInvalid(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Test releasing with missing ID.
	req, _ := http.NewRequest("DELETE", server.URL+"/test", nil)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "missing_id", 400)

	// Test releasing with invalid ID.
	req, _ = http.NewRequest("DELETE", server.URL+"/test?id=abc123", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "invalid_id", 400)

	// Test releasing with invalid path.
	req, _ = http.NewRequest("DELETE", server.URL+"/test/?id=123", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "not_found", 404)

	// Test releasing with ID that is not the locker.
	req, _ = http.NewRequest("DELETE", server.URL+"/test?id=123", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "not_found", 404)
}

func TestHandlerReleaseLocker(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Acquire a ticket.
	ticket, _ := manager.Acquire("test", time.Minute, time.Minute)

	// Test releasing with ID that is not the locker.
	req, _ := http.NewRequest("DELETE", server.URL+fmt.Sprintf("/test?id=%d", ticket.Id()), nil)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertSuccessResponse(t, resp)

	locker, err := manager.IsLocked("test")
	if locker != 0 || err != nil {
		t.Fatalf("Unexpected state after releasing")
	}
}

func TestHandlerExtendInvalid(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Test extending with missing parameters.
	req, _ := http.NewRequest("PATCH", server.URL+"/test?lease_timeout=1m", nil)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "missing_id", 400)

	req, _ = http.NewRequest("PATCH", server.URL+"/test?id=123", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "missing_lease_timeout", 400)

	// Test extending with invalid ID.
	req, _ = http.NewRequest("PATCH", server.URL+"/test?id=abc12&lease_timeout=1m", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "invalid_id", 400)

	// Test extending with invalid lease timeout.
	req, _ = http.NewRequest("PATCH", server.URL+"/test?id=123&lease_timeout=1d", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "invalid_lease_timeout", 400)

	// Test extending with invalid path.
	req, _ = http.NewRequest("PATCH", server.URL+"/test/?id=123&lease_timeout=1m", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "not_found", 404)

	// Test extending with ID that is not the locker.
	req, _ = http.NewRequest("PATCH", server.URL+"/test?id=123&lease_timeout=1m", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "not_found", 404)
}

func TestHandlerExtendLocker(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Acquire a ticket.
	ticket, _ := manager.Acquire("test", time.Minute, time.Minute)
	state, _ := manager.Inspect("test")

	// Test extending with ID that is not the locker.
	req, _ := http.NewRequest("PATCH", server.URL+fmt.Sprintf("/test?id=%d&lease_timeout=5m", ticket.Id()), nil)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertSuccessResponse(t, resp)

	newState, _ := manager.Inspect("test")
	if newState.LockingId != ticket.Id() || newState.LockTimeout <= state.LockTimeout {
		t.Fatalf("Unexpected state after releasing")
	}
}

func TestHandlerInspectInvalid(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Test inspecting with invalid path.
	req, _ := http.NewRequest("GET", server.URL+"/test/", nil)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "not_found", 404)

	// Test inspecting lock that is not locked.
	req, _ = http.NewRequest("GET", server.URL+"/test", nil)
	resp, err = server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	AssertErrorResponse(t, resp, "not_found", 404)
}

func TestHandlerInspectLocked(t *testing.T) {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	defer server.Close()

	// Acquire a ticket.
	ticketA, _ := manager.Acquire("test", time.Minute, time.Minute)
	ticketB, _ := manager.Acquire("test", time.Minute, time.Minute)
	ticketC, _ := manager.Acquire("test", time.Minute, time.Minute)

	// Test extending with ID that is not the locker.
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	body := AssertSuccessResponse(t, resp)
	t.Logf("Inspected body: %v", body)

	if body.LockingId != fmt.Sprintf("%d", ticketA.Id()) {
		t.Fatalf("Expected locking ID to be %d, but it is %s", ticketA.Id(), body.LockingId)
	}
	if body.LockTimeout == "" || body.LockTimeout == "0" {
		t.Fatalf("Unxpected lock timeout: %s", body.LockTimeout)
	}

	if len(body.Acquirers) != 2 {
		t.Fatalf("Expected 2 acquirers in response")
	}

	if body.Acquirers[0].Id != fmt.Sprintf("%d", ticketB.Id()) {
		t.Fatalf("Expected acquirer #1 ID to be %d, but it is %s", ticketB.Id(), body.Acquirers[0].Id)
	}
	if body.Acquirers[0].Timeout == "" || body.Acquirers[0].Timeout == "0" {
		t.Fatalf("Unxpected acquirer #1 timeout: %s", body.Acquirers[0].Timeout)
	}

	if body.Acquirers[1].Id != fmt.Sprintf("%d", ticketC.Id()) {
		t.Fatalf("Expected acquirer #2 ID to be %d, but it is %s", ticketB.Id(), body.Acquirers[1].Id)
	}
	if body.Acquirers[1].Timeout == "" || body.Acquirers[1].Timeout == "0" {
		t.Fatalf("Unxpected acquirer #2 timeout: %s", body.Acquirers[1].Timeout)
	}
}

func Post(t *testing.T, server *httptest.Server, path string, form url.Values) *http.Response {
	resp, err := server.Client().PostForm(server.URL+path, form)
	if err != nil {
		t.Fatalf("Error performing request: %v", err)
	}

	return resp
}

func AssertErrorResponse(t *testing.T, resp *http.Response, code string, statusCode int) {
	if resp.StatusCode != statusCode {
		t.Fatalf("Expected status code %d, got %d", statusCode, resp.StatusCode)
	}

	var body ErrorResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&body); err != nil {
		t.Fatalf("Error decoding response body: %v", err)
	}

	if body.Code != code {
		t.Fatalf("Expected error code %s, got %s", code, body.Code)
	}
}

func AssertSuccessResponse(t *testing.T, resp *http.Response) SuccessResponse {
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status code %d, got %d", 200, resp.StatusCode)
	}

	var body SuccessResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&body); err != nil {
		t.Fatalf("Error decoding response body: %v", err)
	}

	return body
}
