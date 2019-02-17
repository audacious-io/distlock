package httpserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"lockerd/locking"
)

type HandlerFixture struct {
	Manager locking.Manager
	server  *httptest.Server
	t       *testing.T
}

func NewHandlerFixture(t *testing.T) *HandlerFixture {
	manager := locking.NewManager(locking.Config{})
	server := httptest.NewServer(NewHandler(manager))
	manager.Start()

	return &HandlerFixture{
		Manager: manager,
		server:  server,
		t:       t,
	}
}

func (f *HandlerFixture) Close() {
	f.Manager.Stop()
	f.server.Close()
}

func (f *HandlerFixture) Request(method, path string, params url.Values) *http.Response {
	var body io.Reader

	if method == "POST" || method == "PATCH" || method == "PUT" {
		body = strings.NewReader(params.Encode())
		f.t.Logf("Performing %s %s with request body %s", method, path, params.Encode())
	} else {
		if len(params) > 0 {
			path = path + "?" + params.Encode()
		}

		f.t.Logf("Performing %s %s", method, path)
	}

	req, err := http.NewRequest(method, f.server.URL+path, body)
	if err != nil {
		f.t.Fatalf("Error building response: %v", err)
	}

	if body != nil {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := f.server.Client().Do(req)
	if err != nil {
		f.t.Fatalf("Error performing request: %v", err)
	}

	return resp
}
