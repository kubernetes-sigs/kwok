/*
Copyright 2024 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package progressbar

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestNewTransport verifies the NewTransport function.
func TestNewTransport(t *testing.T) {
	// Mock base RoundTripper
	base := &mockRoundTripper{}

	// Create new transport with progress bar
	transport := NewTransport(base)

	// Verify that transport is an http.RoundTripper
	// We can directly use it as an http.RoundTripper
	resp, err := transport.RoundTrip(nil) // Perform a RoundTrip operation to ensure transport is functional
	if err != nil {
		t.Errorf("NewTransport did not return a functional RoundTripper: %v", err)
	}
	// Ensure the response body is closed
	defer func() {
		if resp != nil && resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Error closing response body: %v", err)
			}
		}
	}()
}

// TestTransportRoundTrip verifies the RoundTrip method of the transport.
func TestTransportRoundTrip(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a response with status OK and Content-Length header
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
	}))

	// Close the server when the test finishes
	defer server.Close()

	// Mock base RoundTripper
	base := &mockRoundTripper{}

	// Create new transport with progress bar
	transport := NewTransport(base)

	// Create a sample HTTP request using the test server URL
	req, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("Error creating HTTP request: %v", err)
	}

	// Perform a RoundTrip operation
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Errorf("RoundTrip operation failed: %v", err)
	}

	// Ensure the response body is closed
	defer func() {
		if resp != nil && resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				t.Errorf("Error closing response body: %v", err)
			}
		}
	}()

	// Verify that the response is not nil
	if resp == nil {
		t.Error("RoundTrip did not return a valid response")
	}

	// Verify the status code of the response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Verify the Content-Length header in the response
	contentLength := resp.Header.Get("Content-Length")
	if contentLength == "" {
		t.Error("Content-Length header not found in response")
	}
}

// Mock RoundTripper for testing
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// You can customize the response based on your test requirements
	// For simplicity, we'll include a dummy Content-Length header
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       nil,
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Length", "100")
	return resp, nil
}
