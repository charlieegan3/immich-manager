package addperson

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"immich-manager/pkg/immich"
)

//nolint:maintidx
func TestGenerator_Generate(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/search/metadata" && r.Method == http.MethodPost:
			// Parse request body to get page number
			var searchReq SearchMetadataRequest
			if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)

				return
			}

			// Mock paginated response
			page, _ := strconv.Atoi(searchReq.Page)

			var response SearchMetadataResponse

			switch page {
			case 1:
				// First page with assets and nextPage
				nextPage := "2"
				response = SearchMetadataResponse{
					Assets: struct {
						Items    []Asset `json:"items"`
						NextPage *string `json:"nextPage"`
					}{
						Items: []Asset{
							{ID: "asset1"},
							{ID: "asset2"},
						},
						NextPage: &nextPage,
					},
				}
			case 2:
				// Second page with assets, no nextPage (end of results)
				response = SearchMetadataResponse{
					Assets: struct {
						Items    []Asset `json:"items"`
						NextPage *string `json:"nextPage"`
					}{
						Items: []Asset{
							{ID: "asset3"},
						},
						NextPage: nil,
					},
				}
			default:
				// Should not reach here in our test
				http.Error(w, "Unexpected page", http.StatusBadRequest)

				return
			}

			_ = json.NewEncoder(w).Encode(response)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums") && strings.Contains(r.URL.RawQuery, "assetId="):
			// Mock albums by asset ID
			assetID := r.URL.Query().Get("assetId")

			var albums AlbumsByAssetResponse

			switch assetID {
			case "asset1":
				albums = AlbumsByAssetResponse{
					{ID: "album1"},
					{ID: "album2"},
				}
			case "asset2":
				albums = AlbumsByAssetResponse{
					{ID: "album2"},
					{ID: "album3"},
				}
			case "asset3":
				albums = AlbumsByAssetResponse{
					{ID: "album3"},
				}
			}

			_ = json.NewEncoder(w).Encode(albums)

			return

		case r.URL.Path == "/api/users":
			// Return test users
			users := []immich.User{
				{ID: "user123", Email: "test@example.com", Name: "Test User"},
				{ID: "user456", Email: "other@example.com", Name: "Other User"},
			}
			_ = json.NewEncoder(w).Encode(users)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums/") && !strings.Contains(r.URL.RawQuery, "assetId="):
			// Get album details by ID
			pathParts := strings.Split(r.URL.Path, "/")
			if len(pathParts) < 4 {
				http.Error(w, "Invalid album path", http.StatusBadRequest)

				return
			}

			albumID := pathParts[3]

			// Create a user object for album2 where the user is already a member
			userInAlbum := immich.AlbumUser{}
			userInAlbum.User.ID = "user123" // This matches the test user we'll be looking up
			userInAlbum.Role = "viewer"

			var album immich.Album

			switch albumID {
			case "album1":
				album = immich.Album{ID: "album1", Name: "Summer Vacation", AlbumUsers: []immich.AlbumUser{}}
			case "album2":
				// User is already in this album
				album = immich.Album{ID: "album2", Name: "Work Photos", AlbumUsers: []immich.AlbumUser{userInAlbum}}
			case "album3":
				album = immich.Album{ID: "album3", Name: "Family Photos", AlbumUsers: []immich.AlbumUser{}}
			default:
				http.Error(w, "Album not found", http.StatusNotFound)

				return
			}

			_ = json.NewEncoder(w).Encode(album)

			return

		default:
			w.WriteHeader(http.StatusNotFound)

			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "person123", "test@example.com")

	// Generate plan
	p, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify plan: should have operations for album1 and album3, but not album2 (user already in it)
	expectedOperations := 2
	if len(p.Operations) != expectedOperations {
		t.Errorf("Expected %d operations, got %d", expectedOperations, len(p.Operations))
	}

	// Track which albums were included in the plan
	albumsInPlan := make(map[string]bool)

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
		albumsInPlan[albumID] = true

		// Verify apply method
		if applyReq.Method != http.MethodPut {
			t.Errorf("Expected method PUT, got %s", applyReq.Method)
		}

		// Verify apply body
		var applyBody map[string]any
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
		if revertReq.Method != http.MethodDelete {
			t.Errorf("Expected method DELETE, got %s", revertReq.Method)
		}
	}

	// Verify that album1 and album3 are in the plan (user missing from these)
	if !albumsInPlan["album1"] {
		t.Error("Expected album1 to be in the plan (user should be added)")
	}

	if !albumsInPlan["album3"] {
		t.Error("Expected album3 to be in the plan (user should be added)")
	}

	// Verify that album2 is NOT in the plan (user already in it)
	if albumsInPlan["album2"] {
		t.Error("album2 should not be in the plan (user already in it)")
	}
}

func TestGenerator_NoAssetsForPerson(t *testing.T) {
	t.Parallel()

	// Create test server that returns empty results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/search/metadata":
			// Return empty response
			response := SearchMetadataResponse{
				Assets: struct {
					Items    []Asset `json:"items"`
					NextPage *string `json:"nextPage"`
				}{
					Items:    []Asset{},
					NextPage: nil,
				},
			}

			_ = json.NewEncoder(w).Encode(response)

			return
		default:
			w.WriteHeader(http.StatusNotFound)

			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "nonexistent-person", "test@example.com")

	// Generate plan
	_, err := generator.Generate()

	// Expect an error because no assets found for person
	if err == nil {
		t.Error("Expected error for no assets found for person, got nil")
	}

	expectedErrorMsg := "no assets found for person ID 'nonexistent-person'"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestGenerator_NoAlbumsContainAssets(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/search/metadata":
			// Return assets
			response := SearchMetadataResponse{
				Assets: struct {
					Items    []Asset `json:"items"`
					NextPage *string `json:"nextPage"`
				}{
					Items: []Asset{
						{ID: "asset1"},
					},
					NextPage: nil,
				},
			}

			_ = json.NewEncoder(w).Encode(response)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums") && strings.Contains(r.URL.RawQuery, "assetId="):
			// Return empty albums list for all assets
			albums := AlbumsByAssetResponse{}
			_ = json.NewEncoder(w).Encode(albums)

			return

		default:
			w.WriteHeader(http.StatusNotFound)

			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "person123", "test@example.com")

	// Generate plan
	_, err := generator.Generate()

	// Expect an error because no albums contain the assets
	if err == nil {
		t.Error("Expected error for no albums containing assets, got nil")
	}

	expectedErrorMsg := "no albums found containing assets for person ID 'person123'"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestGenerator_UserNotFound(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/search/metadata":
			// Return assets
			response := SearchMetadataResponse{
				Assets: struct {
					Items    []Asset `json:"items"`
					NextPage *string `json:"nextPage"`
				}{
					Items: []Asset{
						{ID: "asset1"},
					},
					NextPage: nil,
				},
			}

			_ = json.NewEncoder(w).Encode(response)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums") && strings.Contains(r.URL.RawQuery, "assetId="):
			// Return albums containing the asset
			albums := AlbumsByAssetResponse{
				{ID: "album1"},
			}

			_ = json.NewEncoder(w).Encode(albums)

			return

		case r.URL.Path == "/api/users":
			// Return users but none with the email we're looking for
			users := []immich.User{
				{ID: "user456", Email: "other@example.com", Name: "Other User"},
			}

			_ = json.NewEncoder(w).Encode(users)

			return

		default:
			w.WriteHeader(http.StatusNotFound)

			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "person123", "nonexistent@example.com")

	// Generate plan
	_, err := generator.Generate()

	// Expect an error because user doesn't exist
	if err == nil {
		t.Error("Expected error for user not found, got nil")
	}

	expectedErrorMsg := "no user found with email 'nonexistent@example.com'"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestGenerator_UserAlreadyInAllAlbums(t *testing.T) {
	t.Parallel()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/search/metadata":
			// Return assets
			response := SearchMetadataResponse{
				Assets: struct {
					Items    []Asset `json:"items"`
					NextPage *string `json:"nextPage"`
				}{
					Items: []Asset{
						{ID: "asset1"},
					},
					NextPage: nil,
				},
			}

			_ = json.NewEncoder(w).Encode(response)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums") && strings.Contains(r.URL.RawQuery, "assetId="):
			// Return albums containing the asset
			albums := AlbumsByAssetResponse{
				{ID: "album1"},
			}

			_ = json.NewEncoder(w).Encode(albums)

			return

		case r.URL.Path == "/api/users":
			// Return test user
			users := []immich.User{
				{ID: "user123", Email: "test@example.com", Name: "Test User"},
			}

			_ = json.NewEncoder(w).Encode(users)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums/") && !strings.Contains(r.URL.RawQuery, "assetId="):
			// Return album where user is already a member
			userInAlbum := immich.AlbumUser{}
			userInAlbum.User.ID = "user123"
			userInAlbum.Role = "viewer"

			album := immich.Album{
				ID:         "album1",
				Name:       "Test Album",
				AlbumUsers: []immich.AlbumUser{userInAlbum},
			}

			_ = json.NewEncoder(w).Encode(album)

			return

		default:
			w.WriteHeader(http.StatusNotFound)

			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "person123", "test@example.com")

	// Generate plan
	_, err := generator.Generate()

	// Expect an error because user is already in all albums
	if err == nil {
		t.Error("Expected error for user already in all albums, got nil")
	}

	expectedErrorMsg := "no changes needed - user is already in all albums containing assets for this person"
	if !strings.Contains(err.Error(), expectedErrorMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsg, err.Error())
	}
}

func TestGenerator_PaginatedAssets(t *testing.T) {
	t.Parallel()

	requestCount := 0
	// Create test server that tests pagination properly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/search/metadata" && r.Method == http.MethodPost:
			requestCount++

			// Parse request body to get page number
			var searchReq SearchMetadataRequest
			if err := json.NewDecoder(r.Body).Decode(&searchReq); err != nil {
				http.Error(w, "Invalid request body", http.StatusBadRequest)

				return
			}

			// Mock paginated response across 3 pages
			page, _ := strconv.Atoi(searchReq.Page)

			var response SearchMetadataResponse

			switch page {
			case 1:
				nextPage := "2"
				response = SearchMetadataResponse{
					Assets: struct {
						Items    []Asset `json:"items"`
						NextPage *string `json:"nextPage"`
					}{
						Items: []Asset{
							{ID: "asset1"},
							{ID: "asset2"},
						},
						NextPage: &nextPage,
					},
				}
			case 2:
				nextPage := "3"
				response = SearchMetadataResponse{
					Assets: struct {
						Items    []Asset `json:"items"`
						NextPage *string `json:"nextPage"`
					}{
						Items: []Asset{
							{ID: "asset3"},
							{ID: "asset4"},
						},
						NextPage: &nextPage,
					},
				}
			case 3:
				// Final page
				response = SearchMetadataResponse{
					Assets: struct {
						Items    []Asset `json:"items"`
						NextPage *string `json:"nextPage"`
					}{
						Items: []Asset{
							{ID: "asset5"},
						},
						NextPage: nil,
					},
				}
			default:
				http.Error(w, "Unexpected page", http.StatusBadRequest)

				return
			}

			_ = json.NewEncoder(w).Encode(response)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums") && strings.Contains(r.URL.RawQuery, "assetId="):
			// All assets belong to the same album for simplicity
			albums := AlbumsByAssetResponse{
				{ID: "album1"},
			}

			_ = json.NewEncoder(w).Encode(albums)

			return

		case r.URL.Path == "/api/users":
			users := []immich.User{
				{ID: "user123", Email: "test@example.com", Name: "Test User"},
			}

			_ = json.NewEncoder(w).Encode(users)

			return

		case strings.HasPrefix(r.URL.Path, "/api/albums/") && !strings.Contains(r.URL.RawQuery, "assetId="):
			album := immich.Album{
				ID:         "album1",
				Name:       "Test Album",
				AlbumUsers: []immich.AlbumUser{}, // User not in album
			}

			_ = json.NewEncoder(w).Encode(album)

			return

		default:
			w.WriteHeader(http.StatusNotFound)

			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "person123", "test@example.com")

	// Generate plan
	p, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify that we made 3 requests to get all pages
	if requestCount != 3 {
		t.Errorf("Expected 3 requests for pagination, got %d", requestCount)
	}

	// Verify that we got a plan with one operation (all assets in same album)
	if len(p.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(p.Operations))
	}
}
