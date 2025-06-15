package smart

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

func TestGenerator_Generate(t *testing.T) {
	// Set up test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users":
			// Return mock users
			users := []immich.User{
				{ID: "user123", Email: "test@example.com", Name: "Test User"},
				{ID: "user456", Email: "other@example.com", Name: "Other User"},
			}
			json.NewEncoder(w).Encode(users)
			return

		case "/api/albums":
			// Return mock albums
			albums := []immich.Album{
				{
					ID:   "album1",
					Name: "Vacation Photos",
					AlbumUsers: []immich.AlbumUser{
						{
							User: struct {
								ID string `json:"id"`
							}{ID: "user123"},
							Role: "viewer",
						},
					},
				},
				{
					ID:   "album2",
					Name: "Work Photos",
					AlbumUsers: []immich.AlbumUser{
						{
							User: struct {
								ID string `json:"id"`
							}{ID: "user123"},
							Role: "viewer",
						},
					},
				},
				{
					ID:   "album3",
					Name: "Family Photos",
					AlbumUsers: []immich.AlbumUser{
						{
							User: struct {
								ID string `json:"id"`
							}{ID: "user456"},
							Role: "viewer",
						},
					},
				},
				{
					ID:   "smartalbum",
					Name: "All Test User",
					AlbumUsers: []immich.AlbumUser{
						{
							User: struct {
								ID string `json:"id"`
							}{ID: "user123"},
							Role: "owner",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(albums)
			return

		case "/api/albums/album1":
			// Return album1 with assets
			response := struct {
				ID     string `json:"id"`
				Name   string `json:"albumName"`
				Assets []struct {
					ID string `json:"id"`
				} `json:"assets"`
			}{
				ID:   "album1",
				Name: "Vacation Photos",
				Assets: []struct {
					ID string `json:"id"`
				}{
					{ID: "asset1"},
					{ID: "asset2"},
					{ID: "asset3"},
				},
			}
			json.NewEncoder(w).Encode(response)
			return

		case "/api/albums/album2":
			// Return album2 with assets
			response := struct {
				ID     string `json:"id"`
				Name   string `json:"albumName"`
				Assets []struct {
					ID string `json:"id"`
				} `json:"assets"`
			}{
				ID:   "album2",
				Name: "Work Photos",
				Assets: []struct {
					ID string `json:"id"`
				}{
					{ID: "asset3"},
					{ID: "asset4"},
					{ID: "asset5"},
				},
			}
			json.NewEncoder(w).Encode(response)
			return

		case "/api/albums/album3":
			// Return album3 with assets (not shared with our test user)
			response := struct {
				ID     string `json:"id"`
				Name   string `json:"albumName"`
				Assets []struct {
					ID string `json:"id"`
				} `json:"assets"`
			}{
				ID:   "album3",
				Name: "Family Photos",
				Assets: []struct {
					ID string `json:"id"`
				}{
					{ID: "asset6"},
					{ID: "asset7"},
				},
			}
			json.NewEncoder(w).Encode(response)
			return

		case "/api/albums/smartalbum":
			// Return smart album with assets
			response := struct {
				ID     string `json:"id"`
				Name   string `json:"albumName"`
				Assets []struct {
					ID string `json:"id"`
				} `json:"assets"`
			}{
				ID:   "smartalbum",
				Name: "All Test User",
				Assets: []struct {
					ID string `json:"id"`
				}{
					{ID: "asset1"},
					{ID: "asset2"},
					{ID: "asset3"},
					{ID: "asset8"}, // This asset should be removed as it's not in any shared album
				},
			}
			json.NewEncoder(w).Encode(response)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "test@example.com")

	// Generate plan
	p, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify the plan
	verifySmartAlbumPlan(t, p)
}

func verifySmartAlbumPlan(t *testing.T, p *plan.Plan) {
	// The plan should have 2 operations:
	// 1. Add assets 4 and 5 to the smart album
	// 2. Remove asset 8 from the smart album
	if len(p.Operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(p.Operations))
		return
	}

	// Check each operation
	var foundAddOperation, foundRemoveOperation bool

	for _, op := range p.Operations {
		if len(op.Apply) != 1 {
			t.Errorf("Expected 1 apply request per operation, got %d", len(op.Apply))
			continue
		}

		if len(op.Revert) != 1 {
			t.Errorf("Expected 1 revert request per operation, got %d", len(op.Revert))
			continue
		}

		applyReq := op.Apply[0]
		revertReq := op.Revert[0]

		// Check add assets operation
		if applyReq.Method == "PUT" && applyReq.Path == "/api/albums/smartalbum/assets" {
			foundAddOperation = true

			// Verify it's adding the correct assets
			var addBody map[string]interface{}
			if err := json.Unmarshal(applyReq.Body, &addBody); err != nil {
				t.Fatalf("Failed to unmarshal add body: %v", err)
			}

			assetIDs, ok := addBody["ids"].([]interface{})
			if !ok {
				t.Errorf("Expected ids to be an array, got %T", addBody["ids"])
				continue
			}

			// Convert to a map for easier lookup
			addAssetMap := make(map[string]bool)
			for _, id := range assetIDs {
				if idStr, ok := id.(string); ok {
					addAssetMap[idStr] = true
				}
			}

			// Should add asset4 and asset5 which are in album2 but not in smartalbum
			expectedAdds := map[string]bool{"asset4": true, "asset5": true}
			if !reflect.DeepEqual(addAssetMap, expectedAdds) {
				t.Errorf("Expected to add assets %v, got %v", expectedAdds, addAssetMap)
			}

			// Verify revert operation removes the same assets
			if revertReq.Method != "DELETE" || revertReq.Path != "/api/albums/smartalbum/assets" {
				t.Errorf("Expected revert to be DELETE on the same path, got %s %s", revertReq.Method, revertReq.Path)
			}
		}

		// Check remove assets operation
		if applyReq.Method == "DELETE" && applyReq.Path == "/api/albums/smartalbum/assets" {
			foundRemoveOperation = true

			// Verify it's removing the correct assets
			var removeBody map[string]interface{}
			if err := json.Unmarshal(applyReq.Body, &removeBody); err != nil {
				t.Fatalf("Failed to unmarshal remove body: %v", err)
			}

			assetIDs, ok := removeBody["ids"].([]interface{})
			if !ok {
				t.Errorf("Expected ids to be an array, got %T", removeBody["ids"])
				continue
			}

			// Convert to a map for easier lookup
			removeAssetMap := make(map[string]bool)
			for _, id := range assetIDs {
				if idStr, ok := id.(string); ok {
					removeAssetMap[idStr] = true
				}
			}

			// Should remove asset8 which is in smartalbum but not in any shared album
			expectedRemoves := map[string]bool{"asset8": true}
			if !reflect.DeepEqual(removeAssetMap, expectedRemoves) {
				t.Errorf("Expected to remove assets %v, got %v", expectedRemoves, removeAssetMap)
			}

			// Verify revert operation adds the same assets back
			if revertReq.Method != "PUT" || revertReq.Path != "/api/albums/smartalbum/assets" {
				t.Errorf("Expected revert to be PUT on the same path, got %s %s", revertReq.Method, revertReq.Path)
			}
		}
	}

	if !foundAddOperation {
		t.Errorf("No operation found to add missing assets to the smart album")
	}

	if !foundRemoveOperation {
		t.Errorf("No operation found to remove extra assets from the smart album")
	}
}

func TestGenerator_ErrorWhenAlbumDoesNotExist(t *testing.T) {
	// Set up test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/users":
			// Return mock users
			users := []immich.User{
				{ID: "user123", Email: "test@example.com", Name: "Test User"},
			}
			json.NewEncoder(w).Encode(users)
			return

		case r.URL.Path == "/api/albums":
			// Return albums without the smart album
			albums := []immich.Album{
				{
					ID:   "album1",
					Name: "Vacation Photos",
					AlbumUsers: []immich.AlbumUser{
						{
							User: struct {
								ID string `json:"id"`
							}{ID: "user123"},
							Role: "viewer",
						},
					},
				},
				{
					ID:   "album2",
					Name: "Work Photos",
					AlbumUsers: []immich.AlbumUser{
						{
							User: struct {
								ID string `json:"id"`
							}{ID: "user123"},
							Role: "viewer",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(albums)
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "test@example.com")

	// Generate plan - should error because the smart album doesn't exist
	_, err := generator.Generate()
	if err == nil {
		t.Fatalf("Expected error when smart album doesn't exist, got nil")
	}

	// Check error message
	expectedErrMsg := "smart album 'All Test User' does not exist - please create it first"
	if !strings.Contains(err.Error(), expectedErrMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrMsg, err.Error())
	}
}