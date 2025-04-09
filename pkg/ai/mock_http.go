package ai

import (
	"net/http"
)

// A global variable for our tests to replace the HTTP client
var mockDoFunc func(req *http.Request) (*http.Response, error)

// MockTransport implements http.RoundTripper for testing
type MockTransport struct{}

// RoundTrip implements the http.RoundTripper interface
func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if mockDoFunc != nil {
		return mockDoFunc(req)
	}
	return nil, nil
}
