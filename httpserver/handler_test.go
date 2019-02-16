package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"distlock/locking"
)

type SuccessResponse struct {
	Id int64 `json:"id"`
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

	if body.Id == 0 {
		t.Fatalf("Expected to have received an ID")
	}

	locker, _ := manager.IsLocked("test")
	if locker != body.Id {
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
