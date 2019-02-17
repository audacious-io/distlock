package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"
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

type InspectAllResponse map[string]SuccessResponse

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorFixture struct {
	Method             string
	Path               string
	Params             url.Values
	ExpectedCode       string
	ExpectedStatusCode int
}

func TestHandlerAcquireInvalid(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	AssertErrors(f, []ErrorFixture{
		// Missing parameters.
		{
			Method: "POST",
			Path:   "/test",
			Params: url.Values{
				"lock_timeout": []string{"1m"},
			},
			ExpectedCode:       "missing_lease_timeout",
			ExpectedStatusCode: 400,
		},

		{
			Method: "POST",
			Path:   "/test",
			Params: url.Values{
				"lease_timeout": []string{"1m"},
			},
			ExpectedCode:       "missing_lock_timeout",
			ExpectedStatusCode: 400,
		},

		// Invalid parameters.
		{
			Method: "POST",
			Path:   "/test",
			Params: url.Values{
				"lock_timeout":  []string{"1m"},
				"lease_timeout": []string{"1d"},
			},
			ExpectedCode:       "invalid_lease_timeout",
			ExpectedStatusCode: 400,
		},

		{
			Method: "POST",
			Path:   "/test",
			Params: url.Values{
				"lock_timeout":  []string{"123a"},
				"lease_timeout": []string{"1m"},
			},
			ExpectedCode:       "invalid_lock_timeout",
			ExpectedStatusCode: 400,
		},

		// Invalid path.
		{
			Method: "POST",
			Path:   "/test/",
			Params: url.Values{
				"id":            []string{"123"},
				"lease_timeout": []string{"1m"},
			},
			ExpectedCode:       "not_found",
			ExpectedStatusCode: 404,
		},
	})
}

func TestHandlerAcquireSuccessful(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	// Test acquiring successfully.
	resp := f.Request("POST", "/test", url.Values{
		"lock_timeout":  []string{"1m"},
		"lease_timeout": []string{"1m"},
	})
	body := AssertSuccessResponse(t, resp)

	if body.Id == "" {
		t.Fatalf("Expected to have received an ID")
	}
	id, _ := strconv.ParseInt(body.Id, 10, 64)

	locker, _ := f.Manager.IsLocked("test")
	if locker != id {
		t.Fatalf("Expected requestor to be locker")
	}
}

func TestHandlerAcquireTimeout(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	// Acquire up front to cause waiting.
	f.Manager.Acquire("test", time.Minute, time.Minute)

	// Test acquiring causing timeout.
	resp := f.Request("POST", "/test", url.Values{
		"lock_timeout":  []string{"10ms"},
		"lease_timeout": []string{"1m"},
	})
	AssertErrorResponse(t, resp, "timeout", 408)
}

func TestHandlerReleaseInvalid(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	AssertErrors(f, []ErrorFixture{
		// Missing parameters.
		{
			Method:             "DELETE",
			Path:               "/test",
			ExpectedCode:       "missing_id",
			ExpectedStatusCode: 400,
		},

		// Invalid parameters.
		{
			Method: "DELETE",
			Path:   "/test",
			Params: url.Values{
				"id": []string{"123a"},
			},
			ExpectedCode:       "invalid_id",
			ExpectedStatusCode: 400,
		},

		// Invalid path.
		{
			Method: "DELETE",
			Path:   "/test/",
			Params: url.Values{
				"id": []string{"123"},
			},
			ExpectedCode:       "not_found",
			ExpectedStatusCode: 404,
		},

		// With ID that is not the locker.
		{
			Method: "DELETE",
			Path:   "/test",
			Params: url.Values{
				"id": []string{"123"},
			},
			ExpectedCode:       "not_found",
			ExpectedStatusCode: 404,
		},
	})
}

func TestHandlerReleaseLocker(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	// Acquire a ticket.
	ticket, _ := f.Manager.Acquire("test", time.Minute, time.Minute)

	// Test releasing with ID that is not the locker.
	resp := f.Request("DELETE", "/test", url.Values{
		"id": []string{fmt.Sprintf("%d", ticket.Id())},
	})
	AssertSuccessResponse(t, resp)

	locker, err := f.Manager.IsLocked("test")
	if locker != 0 || err != nil {
		t.Fatalf("Unexpected state after releasing")
	}
}

func TestHandlerExtendInvalid(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	AssertErrors(f, []ErrorFixture{
		// Missing parameters.
		{
			Method: "PATCH",
			Path:   "/test",
			Params: url.Values{
				"lease_timeout": []string{"1m"},
			},
			ExpectedCode:       "missing_id",
			ExpectedStatusCode: 400,
		},

		{
			Method: "PATCH",
			Path:   "/test",
			Params: url.Values{
				"id": []string{"123"},
			},
			ExpectedCode:       "missing_lease_timeout",
			ExpectedStatusCode: 400,
		},

		// Invalid parameters.
		{
			Method: "PATCH",
			Path:   "/test",
			Params: url.Values{
				"id":            []string{"123"},
				"lease_timeout": []string{"1d"},
			},
			ExpectedCode:       "invalid_lease_timeout",
			ExpectedStatusCode: 400,
		},

		{
			Method: "PATCH",
			Path:   "/test",
			Params: url.Values{
				"id":            []string{"123a"},
				"lease_timeout": []string{"1m"},
			},
			ExpectedCode:       "invalid_id",
			ExpectedStatusCode: 400,
		},

		// Invalid path.
		{
			Method: "PATCH",
			Path:   "/test/",
			Params: url.Values{
				"id":            []string{"123"},
				"lease_timeout": []string{"1m"},
			},
			ExpectedCode:       "not_found",
			ExpectedStatusCode: 404,
		},

		// With ID that is not the locker.
		{
			Method: "PATCH",
			Path:   "/test",
			Params: url.Values{
				"id":            []string{"123"},
				"lease_timeout": []string{"1m"},
			},
			ExpectedCode:       "not_found",
			ExpectedStatusCode: 404,
		},
	})
}

func TestHandlerExtendLocker(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	// Acquire a ticket.
	ticket, _ := f.Manager.Acquire("test", time.Minute, time.Minute)
	state, _ := f.Manager.Inspect("test")

	// Test extending with ID that is not the locker.
	resp := f.Request("PATCH", "/test", url.Values{
		"id":            []string{fmt.Sprintf("%d", ticket.Id())},
		"lease_timeout": []string{"5m"},
	})
	AssertSuccessResponse(t, resp)

	newState, _ := f.Manager.Inspect("test")
	if newState.LockingId != ticket.Id() || newState.LockTimeout <= state.LockTimeout {
		t.Fatalf("Unexpected state after releasing")
	}
}

func TestHandlerInspectInvalid(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	AssertErrors(f, []ErrorFixture{
		// Invalid path.
		{
			Method:             "GET",
			Path:               "/test/",
			ExpectedCode:       "not_found",
			ExpectedStatusCode: 404,
		},

		// Not locked path.
		{
			Method:             "GET",
			Path:               "/test",
			ExpectedCode:       "not_found",
			ExpectedStatusCode: 404,
		},
	})
}

func TestHandlerInspectLocked(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	// Acquire a ticket.
	ticketA, _ := f.Manager.Acquire("test", time.Minute, time.Minute)
	ticketB, _ := f.Manager.Acquire("test", time.Minute, time.Minute)
	ticketC, _ := f.Manager.Acquire("test", time.Minute, time.Minute)

	// Test extending with ID that is not the locker.
	resp := f.Request("GET", "/test", nil)

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

func TestHandlerInspectAll(t *testing.T) {
	f := NewHandlerFixture(t)
	defer f.Close()

	// Test inspecting with no locks held.
	resp := f.Request("GET", "/", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status code %d, got %d", 200, resp.StatusCode)
	}

	var body InspectAllResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Error decoding response body: %v", err)
	}

	if len(body) != 0 {
		t.Fatalf("Expected no locks to be returned")
	}

	// Test inspecting with locks held.
	ticketA, _ := f.Manager.Acquire("a", time.Minute, time.Minute)
	ticketB, _ := f.Manager.Acquire("a", time.Minute, time.Minute)
	ticketC, _ := f.Manager.Acquire("a", time.Minute, time.Minute)
	ticketD, _ := f.Manager.Acquire("b", time.Minute, time.Minute)

	resp = f.Request("GET", "/", nil)
	if resp.StatusCode != 200 {
		t.Fatalf("Expected status code %d, got %d", 200, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Error decoding response body: %v", err)
	}

	if len(body) != 2 {
		t.Fatalf("Expected 2 locks to be returned")
	}

	lock := body["a"]

	if lock.LockingId != fmt.Sprintf("%d", ticketA.Id()) {
		t.Fatalf("Expected locking ID to be %d, but it is %s", ticketA.Id(), lock.LockingId)
	}
	if lock.LockTimeout == "" || lock.LockTimeout == "0" {
		t.Fatalf("Unxpected lock timeout: %s", lock.LockTimeout)
	}

	if len(lock.Acquirers) != 2 {
		t.Fatalf("Expected 2 acquirers in response")
	}

	if lock.Acquirers[0].Id != fmt.Sprintf("%d", ticketB.Id()) {
		t.Fatalf("Expected acquirer #1 ID to be %d, but it is %s", ticketB.Id(), lock.Acquirers[0].Id)
	}
	if lock.Acquirers[0].Timeout == "" || lock.Acquirers[0].Timeout == "0" {
		t.Fatalf("Unxpected acquirer #1 timeout: %s", lock.Acquirers[0].Timeout)
	}

	if lock.Acquirers[1].Id != fmt.Sprintf("%d", ticketC.Id()) {
		t.Fatalf("Expected acquirer #2 ID to be %d, but it is %s", ticketB.Id(), lock.Acquirers[1].Id)
	}
	if lock.Acquirers[1].Timeout == "" || lock.Acquirers[1].Timeout == "0" {
		t.Fatalf("Unxpected acquirer #2 timeout: %s", lock.Acquirers[1].Timeout)
	}

	lock = body["b"]

	if lock.LockingId != fmt.Sprintf("%d", ticketD.Id()) {
		t.Fatalf("Expected locking ID to be %d, but it is %s", ticketD.Id(), lock.LockingId)
	}
	if lock.LockTimeout == "" || lock.LockTimeout == "0" {
		t.Fatalf("Unxpected lock timeout: %s", lock.LockTimeout)
	}

	if len(lock.Acquirers) != 0 {
		t.Fatalf("Expected 0 acquirers in response")
	}
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

func AssertErrors(f *HandlerFixture, fixtures []ErrorFixture) {
	for _, fix := range fixtures {
		resp := f.Request(fix.Method, fix.Path, fix.Params)
		AssertErrorResponse(f.t, resp, fix.ExpectedCode, fix.ExpectedStatusCode)
	}
}
