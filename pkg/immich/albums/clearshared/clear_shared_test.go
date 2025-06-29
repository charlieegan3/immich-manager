package clearshared

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"immich-manager/pkg/immich"
)

func TestGenerator_Generate(t *testing.T) {
	t.Parallel()

	// Test cases
	tests := []struct {
		name       string
		email      string
		wantErr    bool
		wantAlbums int
	}{
		{
			name:       "User found with shared albums",
			email:      "user2@example.com",
			wantErr:    false,
			wantAlbums: 2, // Both album1 and album2 are shared with user2
		},
		{
			name:       "User not found",
			email:      "nonexistent@example.com",
			wantErr:    true,
			wantAlbums: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle shared albums request
				if r.URL.Path == "/api/albums" && r.Method == http.MethodGet && strings.Contains(r.URL.RawQuery, "shared=true") {
					type AlbumUserUser struct {
						ID    string `json:"id"`
						Email string `json:"email"`
						Name  string `json:"name"`
					}

					type AlbumUser struct {
						User AlbumUserUser `json:"user"`
						Role string        `json:"role"`
					}

					type Owner struct {
						ID               string `json:"id"`
						Email            string `json:"email"`
						Name             string `json:"name"`
						ProfileImagePath string `json:"profileImagePath"`
						AvatarColor      string `json:"avatarColor"`
					}

					type Album struct {
						ID          string      `json:"id"`
						AlbumName   string      `json:"albumName"`
						OwnerID     string      `json:"ownerId"`
						Owner       Owner       `json:"owner"`
						Shared      bool        `json:"shared"`
						AlbumUsers  []AlbumUser `json:"albumUsers"`
						Description string      `json:"description"`
						CreatedAt   string      `json:"createdAt"`
						UpdatedAt   string      `json:"updatedAt"`
						AssetCount  int         `json:"assetCount"`
					}

					owner := Owner{
						ID:    "user1",
						Email: "user1@example.com",
						Name:  "User One",
					}

					user1 := AlbumUserUser{
						ID:    "user1",
						Email: "user1@example.com",
						Name:  "User One",
					}

					user2 := AlbumUserUser{
						ID:    "user2",
						Email: "user2@example.com",
						Name:  "User Two",
					}

					albums := []Album{
						{
							ID:        "album1",
							AlbumName: "Family Photos",
							OwnerID:   "user1",
							Owner:     owner,
							Shared:    true,
							AlbumUsers: []AlbumUser{
								{User: user1, Role: "owner"},
								{User: user2, Role: "viewer"},
							},
						},
						{
							ID:        "album2",
							AlbumName: "Vacation 2023",
							OwnerID:   "user1",
							Owner:     owner,
							Shared:    true,
							AlbumUsers: []AlbumUser{
								{User: user1, Role: "owner"},
								{User: user2, Role: "viewer"},
							},
						},
						{
							ID:        "album3",
							AlbumName: "Work Documents",
							OwnerID:   "user1",
							Owner:     owner,
							Shared:    true,
							AlbumUsers: []AlbumUser{
								{User: user1, Role: "owner"},
							},
						},
					}
					_ = json.NewEncoder(w).Encode(albums)

					return
				}

				// Unexpected request
				w.WriteHeader(http.StatusNotFound)
			}))
			defer server.Close()

			client := immich.NewClient(server.URL, "fake-token")
			generator := NewGenerator(client, tt.email)

			plan, err := generator.Generate()

			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if err != nil {
				return
			}

			if len(plan.Operations) != tt.wantAlbums {
				t.Errorf("Generate() got %d operations, want %d", len(plan.Operations), tt.wantAlbums)
			}

			// Verify operations are correctly formed
			for _, op := range plan.Operations {
				// Check apply operation is a DELETE
				if len(op.Apply) != 1 {
					t.Errorf("Expected 1 apply operation, got %d", len(op.Apply))
				} else if op.Apply[0].Method != http.MethodDelete {
					t.Errorf("Expected DELETE method, got %s", op.Apply[0].Method)
				}

				// Check revert operation is a PUT
				if len(op.Revert) != 1 {
					t.Errorf("Expected 1 revert operation, got %d", len(op.Revert))
				} else if op.Revert[0].Method != http.MethodPut {
					t.Errorf("Expected PUT method, got %s", op.Revert[0].Method)
				}
			}
		})
	}
}
