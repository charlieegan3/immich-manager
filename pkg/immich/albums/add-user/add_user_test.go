package adduser

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"immich-manager/pkg/immich"
)

func TestGenerator_Generate(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/albums":
			// Create a user object for album 3 where the user is already a member
			userInAlbum := immich.AlbumUser{}
			userInAlbum.User.ID = "user123" // This matches the test user we'll be looking up
			userInAlbum.Role = "viewer"

			// Return mock albums
			albums := []immich.Album{
				{ID: "1", Name: "vacation photos", AlbumUsers: []immich.AlbumUser{}},
				{ID: "2", Name: "work photos", AlbumUsers: []immich.AlbumUser{}},
				{ID: "3", Name: "vacation memories", AlbumUsers: []immich.AlbumUser{userInAlbum}},
			}
			json.NewEncoder(w).Encode(albums)
			return
		case "/api/users":
			// Return all users
			users := []immich.User{
				{ID: "user123", Email: "test@example.com", Name: "Test User"},
				{ID: "user456", Email: "other@example.com", Name: "Other User"},
			}
			json.NewEncoder(w).Encode(users)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "vacation", "test@example.com")

	// Generate plan
	p, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify plan
	if len(p.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(p.Operations))
	}

	// Verify operations for albums with "vacation" in the name
	// Album 3 should be excluded because the user is already a member
	expectedAlbumIDs := map[string]bool{"1": false}

	for _, op := range p.Operations {
		// Verify apply operation
		if len(op.Apply) != 1 {
			t.Errorf("Expected 1 apply request, got %d", len(op.Apply))
			continue
		}

		applyReq := op.Apply[0]

		// Extract album ID from path
		// Format is /api/albums/{id}/users
		pathParts := strings.Split(applyReq.Path, "/")
		// The split result has an empty string at the beginning
		if len(pathParts) < 5 {
			t.Errorf("Invalid path format (too short): %s", applyReq.Path)
			continue
		}

		// Path should be: ["", "api", "albums", "{id}", "users"]
		if pathParts[1] != "api" || pathParts[2] != "albums" || pathParts[4] != "users" {
			t.Errorf("Invalid path format (wrong segments): %s", applyReq.Path)
			continue
		}

		albumID := pathParts[3]

		// Mark album as found
		if _, exists := expectedAlbumIDs[albumID]; exists {
			expectedAlbumIDs[albumID] = true
		} else {
			t.Errorf("Unexpected album ID in plan: %s", albumID)
		}

		// Verify apply method
		if applyReq.Method != "PUT" {
			t.Errorf("Expected method PUT, got %s", applyReq.Method)
		}

		// Verify apply body
		var applyBody map[string]interface{}
		if err := json.Unmarshal(applyReq.Body, &applyBody); err != nil {
			t.Fatalf("Failed to unmarshal body: %v", err)
		}

		albumUsers, ok := applyBody["albumUsers"].([]interface{})
		if !ok || len(albumUsers) != 1 {
			t.Errorf("Expected albumUsers array with 1 element, got %v", applyBody["albumUsers"])
			continue
		}

		user, ok := albumUsers[0].(map[string]interface{})
		if !ok {
			t.Errorf("Expected user to be a map, got %v", albumUsers[0])
			continue
		}

		if user["userId"] != "user123" {
			t.Errorf("Expected userId 'user123', got %v", user["userId"])
		}

		if user["role"] != "viewer" {
			t.Errorf("Expected role 'viewer', got %v", user["role"])
		}

		// Verify revert operation
		if len(op.Revert) != 1 {
			t.Errorf("Expected 1 revert request, got %d", len(op.Revert))
			continue
		}

		revertReq := op.Revert[0]

		// Verify revert path
		expectedRevertPath := fmt.Sprintf("/api/albums/%s/user/user123", albumID)
		if revertReq.Path != expectedRevertPath {
			t.Errorf("Expected revert path %s, got %s", expectedRevertPath, revertReq.Path)
		}

		// Verify revert method
		if revertReq.Method != "DELETE" {
			t.Errorf("Expected method DELETE, got %s", revertReq.Method)
		}
	}

	// Check that all expected albums were found
	for albumID, found := range expectedAlbumIDs {
		if !found {
			t.Errorf("Album %s was not included in the plan", albumID)
		}
	}
}

func TestGenerator_NoMatchingAlbums(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/albums":
			// Return mock albums with no matches
			albums := []immich.Album{
				{ID: "1", Name: "work photos", AlbumUsers: []immich.AlbumUser{}},
				{ID: "2", Name: "family photos", AlbumUsers: []immich.AlbumUser{}},
			}
			json.NewEncoder(w).Encode(albums)
			return
		case "/api/users":
			// Return a mock user
			users := []immich.User{
				{ID: "user123", Email: "test@example.com", Name: "Test User"},
			}
			json.NewEncoder(w).Encode(users)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "vacation", "test@example.com")

	// Generate plan
	_, err := generator.Generate()

	// Expect an error because no albums match
	if err == nil {
		t.Error("Expected error for no matching albums, got nil")
	}
}

func TestGenerator_UserNotFound(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/albums":
			// Return mock albums
			albums := []immich.Album{
				{ID: "1", Name: "vacation photos", AlbumUsers: []immich.AlbumUser{}},
			}
			json.NewEncoder(w).Encode(albums)
			return
		case "/api/users":
			// Return users but none with the email we're looking for
			users := []immich.User{
				{ID: "user456", Email: "other@example.com", Name: "Other User"},
			}
			json.NewEncoder(w).Encode(users)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "vacation", "nonexistent@example.com")

	// Generate plan
	_, err := generator.Generate()

	// Expect an error because user doesn't exist
	if err == nil {
		t.Error("Expected error for user not found, got nil")
	}
}

