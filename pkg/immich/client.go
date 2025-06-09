package immich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"immich-manager/pkg/immich/types"
)

// Client represents an Immich API client
type Client struct {
	serverURL string
	token     string
	client    *http.Client
}

// NewClient creates a new Immich API client
func NewClient(serverURL, token string) *Client {
	return &Client{
		serverURL: strings.TrimRight(serverURL, "/"),
		token:     token,
		client:    &http.Client{},
	}
}

// ServerURL returns the server URL
func (c *Client) ServerURL() string {
	return c.serverURL
}

// Token returns the authentication token
func (c *Client) Token() string {
	return c.token
}

// Client returns the HTTP client
func (c *Client) Client() *http.Client {
	return c.client
}

// SetAuthHeader sets the Immich API authentication header
func (c *Client) SetAuthHeader(req *http.Request) {
	req.Header.Set("x-api-key", c.token)
}

// NewRequest creates a new HTTP request with the given method and path
func (c *Client) NewRequest(method, path string, body interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.serverURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.SetAuthHeader(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// Do performs the HTTP request and decodes the response into the provided value
func (c *Client) Do(req *http.Request, v interface{}) error {
	// Save the request body for error reporting
	var requestBodyBytes []byte
	var requestBodyCopy bytes.Buffer
	
	if req.Body != nil {
		// Create a copy of the request body
		bodyReader := io.TeeReader(req.Body, &requestBodyCopy)
		requestBodyBytes, _ = io.ReadAll(bodyReader)
		
		// Reset the request body with the original content
		req.Body = io.NopCloser(bytes.NewBuffer(requestBodyBytes))
	}
	
	// Execute the request
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()
	
	// Read the entire response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}
	
	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Create detailed error message with request and response information
		var requestBodyStr string
		if len(requestBodyBytes) > 0 {
			requestBodyStr = string(requestBodyBytes)
		} else {
			requestBodyStr = "<empty>"
		}
		
		return fmt.Errorf("API error: %s %s\nStatus: %d %s\nRequest body: %s\nResponse body: %s",
			req.Method, req.URL.String(),
			resp.StatusCode, resp.Status,
			requestBodyStr,
			string(respBody))
	}
	
	// Reset response body for further processing
	resp.Body = io.NopCloser(bytes.NewBuffer(respBody))
	
	// Decode the response if needed
	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("decoding response: %w\nResponse body: %s", err, string(respBody))
		}
	}
	
	return nil
}

// Album represents an Immich album
type Album = types.Album

// User represents an Immich user
type User = types.User

// AlbumUser represents a user shared with an album
type AlbumUser = types.AlbumUser
