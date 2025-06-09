package immich

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if v != nil {
		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// Album represents an Immich album
type Album struct {
	ID   string `json:"id"`
	Name string `json:"albumName"`
}
