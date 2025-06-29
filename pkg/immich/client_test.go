package immich

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_Do_ErrorHandling(t *testing.T) {
	t.Parallel()
	// Test cases for different error scenarios
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   map[string]any
		requestBody    map[string]any
		expectedErrMsg []string // Substrings that should appear in the error message
	}{
		{
			name:       "400 Bad Request",
			statusCode: http.StatusBadRequest,
			responseBody: map[string]any{
				"error":   "Bad Request",
				"message": "Invalid album ID format",
			},
			requestBody: map[string]any{
				"albumUsers": []map[string]string{
					{"userId": "invalid-id", "role": "viewer"},
				},
			},
			expectedErrMsg: []string{
				"Status: 400",
				"Invalid album ID format",
				"invalid-id",
			},
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			responseBody: map[string]any{
				"error":   "Unauthorized",
				"message": "Invalid API key",
			},
			requestBody: nil,
			expectedErrMsg: []string{
				"Status: 401",
				"Invalid API key",
			},
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			responseBody: map[string]any{
				"error":   "Forbidden",
				"message": "User does not have permission to modify this album",
			},
			requestBody: map[string]any{
				"albumUsers": []map[string]string{
					{"userId": "user-123", "role": "admin"},
				},
			},
			expectedErrMsg: []string{
				"Status: 403",
				"User does not have permission",
				"user-123",
			},
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			responseBody: map[string]any{
				"error":   "Not Found",
				"message": "Album with ID 'album-456' not found",
			},
			requestBody: nil,
			expectedErrMsg: []string{
				"Status: 404",
				"Album with ID 'album-456' not found",
			},
		},
		{
			name:       "409 Conflict",
			statusCode: http.StatusConflict,
			responseBody: map[string]any{
				"error":   "Conflict",
				"message": "User is already in this album",
			},
			requestBody: map[string]any{
				"albumUsers": []map[string]string{
					{"userId": "user-789", "role": "viewer"},
				},
			},
			expectedErrMsg: []string{
				"Status: 409",
				"User is already in this album",
				"user-789",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// Set the status code
				w.WriteHeader(tc.statusCode)
				// Return the error response
				_ = json.NewEncoder(w).Encode(tc.responseBody)
			}))
			defer server.Close()

			// Create client
			client := NewClient(server.URL, "test-token")

			// Create request with test body
			req, err := client.NewRequest(http.MethodPost, "/api/test-endpoint", tc.requestBody)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Execute the request
			err = client.Do(req, nil)

			// Verify error handling
			if err == nil {
				t.Fatalf("Expected error for status %d, got nil", tc.statusCode)
			}

			// Check that error message contains all expected substrings
			errMsg := err.Error()
			for _, expectedSubstr := range tc.expectedErrMsg {
				if !strings.Contains(errMsg, expectedSubstr) {
					t.Errorf("Error message does not contain expected substring: %s\nActual error: %s",
						expectedSubstr, errMsg)
				}
			}

			// Verify that the error message includes the request URL
			if !strings.Contains(errMsg, "/api/test-endpoint") {
				t.Errorf("Error message does not contain the request URL")
			}

			// If there was a request body, verify it's included in the error
			if tc.requestBody != nil {
				requestBodyJSON, _ := json.Marshal(tc.requestBody)
				requestBodyStr := string(requestBodyJSON)

				if !strings.Contains(errMsg, requestBodyStr) {
					t.Errorf("Error message does not contain the request body: %s", requestBodyStr)
				}
			}

			// Verify that response body is included in the error
			responseBodyJSON, _ := json.Marshal(tc.responseBody)
			responseBodyStr := string(responseBodyJSON)

			if !strings.Contains(errMsg, strings.TrimSpace(responseBodyStr)) {
				t.Errorf("Error message does not contain the response body details.\nExpected substring: %s\nActual error: %s",
					responseBodyStr, errMsg)
			}
		})
	}
}

func TestClient_Do_Success(t *testing.T) {
	t.Parallel()
	// Create test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Return successful response
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":   "album-123",
			"name": "Test Album",
		})
	}))
	defer server.Close()

	// Create client
	client := NewClient(server.URL, "test-token")

	// Create request
	req, err := client.NewRequest(http.MethodGet, "/api/albums/album-123", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Execute the request
	var result map[string]string
	err = client.Do(req, &result)
	// Verify no error
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify result was decoded correctly
	if result["id"] != "album-123" || result["name"] != "Test Album" {
		t.Errorf("Unexpected result: %+v", result)
	}
}
