package applier

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	return data
}

func TestApplier_ApplyAndRevert(t *testing.T) {
	t.Parallel()
	// Create mock state to track album names
	albumState := map[string]string{
		"1": "old name 1",
		"2": "old name 2",
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract album ID from path
		albumID := r.URL.Path[len("/api/albums/"):]

		// Verify request body
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		if body["albumName"] == "" {
			t.Error("Expected albumName in request body")
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		// Update state
		albumState[albumID] = body["albumName"]

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create test plan
	p := &plan.Plan{
		Operations: []plan.Operation{
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/1",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"albumName": "new name 1"}`),
					},
				},
				Revert: []plan.Request{
					{
						Path:   "/api/albums/1",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"albumName": "old name 1"}`),
					},
				},
			},
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/2",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"albumName": "new name 2"}`),
					},
				},
				Revert: []plan.Request{
					{
						Path:   "/api/albums/2",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"albumName": "old name 2"}`),
					},
				},
			},
		},
	}

	// Create client and applier
	client := immich.NewClient(server.URL, "test-token")
	applier := NewApplier(client)

	// Test Apply
	if err := applier.Apply(p, nil); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify final state after apply
	if albumState["1"] != "new name 1" {
		t.Errorf("Expected album 1 name to be 'new name 1', got %s", albumState["1"])
	}

	if albumState["2"] != "new name 2" {
		t.Errorf("Expected album 2 name to be 'new name 2', got %s", albumState["2"])
	}

	// Test Revert
	if err := applier.Revert(p, nil); err != nil {
		t.Fatalf("Revert() error = %v", err)
	}

	// Verify final state after revert
	if albumState["1"] != "old name 1" {
		t.Errorf("Expected album 1 name to be 'old name 1', got %s", albumState["1"])
	}

	if albumState["2"] != "old name 2" {
		t.Errorf("Expected album 2 name to be 'old name 2', got %s", albumState["2"])
	}
}

func TestApplier_ErrorHandling(t *testing.T) {
	t.Parallel()
	// Create test server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create test plan with invalid operation
	p := &plan.Plan{
		Operations: []plan.Operation{
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/1",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"invalid": "json"`), // Invalid JSON
					},
				},
				Revert: []plan.Request{
					{
						Path:   "/api/albums/1",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"albumName": "old name 1"}`),
					},
				},
			},
		},
	}

	// Create client and applier
	client := immich.NewClient(server.URL, "test-token")
	applier := NewApplier(client)

	// Test Apply error
	err := applier.Apply(p, nil)
	if err == nil {
		t.Error("Expected error from Apply(), got nil")
	}

	// Test Revert error
	err = applier.Revert(p, nil)
	if err == nil {
		t.Error("Expected error from Revert(), got nil")
	}
}

// setupDryRunTest creates a test plan and mock server for dry run tests.
func setupDryRunTest(t *testing.T) (*plan.Plan, *Applier) {
	t.Helper()
	// Create test plan
	p := &plan.Plan{
		Operations: []plan.Operation{
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/1",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"albumName": "new name 1"}`),
					},
				},
				Revert: []plan.Request{
					{
						Path:   "/api/albums/1",
						Method: http.MethodPatch,
						Body:   json.RawMessage(`{"albumName": "old name 1"}`),
					},
				},
			},
		},
	}

	// Create a mock server that should NOT be called
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("Server was called during a dry run")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	// Create client and applier
	client := immich.NewClient(server.URL, "test-token")
	applier := NewApplier(client)

	return p, applier
}

func TestDryRunApply(t *testing.T) {
	t.Parallel()
	p, applier := setupDryRunTest(t)

	// Test dry run apply
	var buf bytes.Buffer
	opts := &ApplyOptions{
		DryRun: true,
		Writer: &buf,
	}

	err := applier.Apply(p, opts)
	if err != nil {
		t.Fatalf("Apply() with dry run error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Dry run apply produced no output")
	}

	// Check for expected output strings
	expectedStrings := []string{
		"Dry run mode",
		"would execute 1 operations",
		"Operation 1:",
		"Request 1.1: PATCH /api/albums/1",
		"albumName",
		"new name 1",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("Dry run output missing expected string: %s", s)
		}
	}
}

func TestDryRunRevert(t *testing.T) {
	t.Parallel()
	p, applier := setupDryRunTest(t)

	// Test dry run revert
	var buf bytes.Buffer
	opts := &ApplyOptions{
		DryRun: true,
		Writer: &buf,
	}

	err := applier.Revert(p, opts)
	if err != nil {
		t.Fatalf("Revert() with dry run error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Dry run revert produced no output")
	}

	// Check for expected output strings
	expectedStrings := []string{
		"Dry run mode",
		"would revert 1 operations",
		"Operation 1:",
		"Request 1.1: PATCH /api/albums/1",
		"albumName",
		"old name 1",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(output, s) {
			t.Errorf("Dry run output missing expected string: %s", s)
		}
	}
}

func TestNilBodyRequest(t *testing.T) {
	t.Parallel()
	// Track if the Content-Type header was set
	var contentTypeWasSet bool

	var bodyWasEmpty bool

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Content-Type header
		contentTypeWasSet = r.Header.Get("Content-Type") != ""

		// Check if the body is empty
		body, _ := io.ReadAll(r.Body)
		bodyWasEmpty = len(body) == 0

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create test plan with a DELETE request with no body field
	p := &plan.Plan{
		Operations: []plan.Operation{
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/1/user/123",
						Method: http.MethodDelete,
						// No Body field
					},
				},
				Revert: []plan.Request{
					{
						Path:   "/api/albums/1/users",
						Method: http.MethodPut,
						Body:   json.RawMessage(`{"albumUsers": [{"role": "viewer", "userId": "123"}]}`),
					},
				},
			},
		},
	}

	// Create client and applier
	client := immich.NewClient(server.URL, "test-token")
	applier := NewApplier(client)

	// Test Apply with no body field
	if err := applier.Apply(p, nil); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Verify that Content-Type header was NOT set and body was empty
	if contentTypeWasSet {
		t.Error("Content-Type header was set for request without body field")
	}

	if !bodyWasEmpty {
		t.Error("Request body was not empty for request without body field")
	}
}

func TestDeleteRequestWithBody(t *testing.T) {
	t.Parallel()
	// Track request details
	var receivedMethod string

	var receivedBody map[string]any

	var contentTypeWasSet bool

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		contentTypeWasSet = r.Header.Get("Content-Type") == "application/json"

		// Read and parse the body
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			_ = json.Unmarshal(body, &receivedBody)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test DELETE request with body
	testBody := map[string]any{
		"ids": []string{"asset-id-1", "asset-id-2"},
	}
	testBodyJSON := mustMarshal(testBody)

	// Create test plan with DELETE request that has a body
	p := &plan.Plan{
		Operations: []plan.Operation{
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/test-album-id/assets",
						Method: http.MethodDelete,
						Body:   testBodyJSON,
					},
				},
			},
		},
	}

	// Create applier and apply plan
	client := immich.NewClient(server.URL, "test-token")
	applier := NewApplier(client)

	opts := &ApplyOptions{
		DryRun: false,
		Writer: io.Discard,
	}

	err := applier.Apply(p, opts)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify the request was sent correctly
	if receivedMethod != http.MethodDelete {
		t.Errorf("Expected DELETE method, got %s", receivedMethod)
	}

	if !contentTypeWasSet {
		t.Error("Content-Type header was not set for DELETE request with body")
	}

	if receivedBody == nil {
		t.Fatal("Request body was not received")
	}

	// Verify body content
	ids, ok := receivedBody["ids"].([]interface{})
	if !ok {
		t.Fatal("Body does not contain 'ids' array")
	}

	if len(ids) != 2 {
		t.Errorf("Expected 2 asset IDs, got %d", len(ids))
	}

	id0, ok0 := ids[0].(string)
	id1, ok1 := ids[1].(string)

	if !ok0 || !ok1 || id0 != "asset-id-1" || id1 != "asset-id-2" {
		t.Errorf("Asset IDs don't match: got %v", ids)
	}
}

func TestDeleteRequestWithoutBody(t *testing.T) {
	t.Parallel()
	// Track request details
	var receivedMethod string

	var bodyWasEmpty bool

	var contentTypeWasSet bool

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		contentTypeWasSet = r.Header.Get("Content-Type") != ""

		// Check if the body is empty
		body, _ := io.ReadAll(r.Body)
		bodyWasEmpty = len(body) == 0

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create test plan with DELETE request without body
	p := &plan.Plan{
		Operations: []plan.Operation{
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/test-album-id/user/123",
						Method: http.MethodDelete,
						// No Body field
					},
				},
			},
		},
	}

	// Create applier and apply plan
	client := immich.NewClient(server.URL, "test-token")
	applier := NewApplier(client)

	opts := &ApplyOptions{
		DryRun: false,
		Writer: io.Discard,
	}

	err := applier.Apply(p, opts)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify the request was sent correctly
	if receivedMethod != http.MethodDelete {
		t.Errorf("Expected DELETE method, got %s", receivedMethod)
	}

	if contentTypeWasSet {
		t.Error("Content-Type header should not be set for DELETE request without body")
	}

	if !bodyWasEmpty {
		t.Error("Request body should be empty for DELETE request without body")
	}
}

func TestDeleteRequestWithNilBody(t *testing.T) {
	t.Parallel()
	// Track request details
	var receivedMethod string

	var bodyWasEmpty bool

	var contentTypeWasSet bool

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		contentTypeWasSet = r.Header.Get("Content-Type") != ""

		// Check if the body is empty
		body, _ := io.ReadAll(r.Body)
		bodyWasEmpty = len(body) == 0

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Test with nil body (explicit)
	nilBodyJSON := mustMarshal(nil)

	// Create test plan with DELETE request with explicit nil body
	p := &plan.Plan{
		Operations: []plan.Operation{
			{
				Apply: []plan.Request{
					{
						Path:   "/api/albums/test-album-id/user/123",
						Method: http.MethodDelete,
						Body:   nilBodyJSON,
					},
				},
			},
		},
	}

	// Create applier and apply plan
	client := immich.NewClient(server.URL, "test-token")
	applier := NewApplier(client)

	opts := &ApplyOptions{
		DryRun: false,
		Writer: io.Discard,
	}

	err := applier.Apply(p, opts)
	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}

	// Verify the request was sent correctly
	if receivedMethod != http.MethodDelete {
		t.Errorf("Expected DELETE method, got %s", receivedMethod)
	}

	if contentTypeWasSet {
		t.Error("Content-Type header should not be set for DELETE request with nil body")
	}

	if !bodyWasEmpty {
		t.Error("Request body should be empty for DELETE request with nil body")
	}
}
