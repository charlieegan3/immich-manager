package clearshared

import (
	"encoding/json"
	"fmt"
	"strings"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

// Generator generates a plan for removing a user from all shared albums
type Generator struct {
	client *immich.Client
	email  string
}

// NewGenerator creates a new plan generator for removing a user from shared albums
func NewGenerator(client *immich.Client, email string) *Generator {
	return &Generator{
		client: client,
		email:  email,
	}
}

// Generate creates a plan for removing a user from shared albums
func (g *Generator) Generate() (*plan.Plan, error) {
	// Get all shared albums
	req, err := g.client.NewRequest("GET", "/api/albums?shared=true", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for shared albums: %w", err)
	}

	// Define structs to match the API response structure
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
		OwnerId     string      `json:"ownerId"`
		Owner       Owner       `json:"owner"`
		Shared      bool        `json:"shared"`
		AlbumUsers  []AlbumUser `json:"albumUsers"`
		Description string      `json:"description"`
		CreatedAt   string      `json:"createdAt"`
		UpdatedAt   string      `json:"updatedAt"`
		AssetCount  int         `json:"assetCount"`
	}

	var sharedAlbums []Album
	if err := g.client.Do(req, &sharedAlbums); err != nil {
		return nil, fmt.Errorf("getting shared albums: %w", err)
	}

	// Filter shared albums by those that include the target user
	userSharedAlbums := make([]Album, 0)
	var targetUserID string

	for _, album := range sharedAlbums {
		// Check if user is in this album
		for _, albumUser := range album.AlbumUsers {
			if strings.EqualFold(albumUser.User.Email, g.email) {
				userSharedAlbums = append(userSharedAlbums, album)
				targetUserID = albumUser.User.ID
				break
			}
		}
	}

	if len(userSharedAlbums) == 0 {
		return nil, fmt.Errorf("no shared albums found for user '%s'", g.email)
	}

	if targetUserID == "" {
		return nil, fmt.Errorf("could not determine user ID for email '%s'", g.email)
	}

	p := &plan.Plan{
		Operations: make([]plan.Operation, 0, len(userSharedAlbums)),
	}

	// Create operations for each album
	for _, album := range userSharedAlbums {
		// Find the user's role in the album for the revert operation
		var userRole string
		for _, albumUser := range album.AlbumUsers {
			if strings.EqualFold(albumUser.User.Email, g.email) {
				userRole = albumUser.Role
				break
			}
		}

		// Default to "viewer" if role is not found or invalid
		if userRole == "" {
			userRole = "viewer"
		} else if userRole != "viewer" && userRole != "editor" {
			// Ensure role is one of the valid values
			userRole = "viewer"
		}

		// Prepare revert (add user back) body
		revertBody := map[string]interface{}{
			"albumUsers": []map[string]string{
				{
					"role":   userRole,
					"userId": targetUserID,
				},
			},
		}
		revertBodyJSON, err := json.Marshal(revertBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling revert body: %w", err)
		}

		// For DELETE requests to the Immich API, we need to explicitly specify
		// nil for the body to ensure no data is sent
		p.Operations = append(p.Operations, plan.Operation{
			Apply: []plan.Request{
				{
					Path:   fmt.Sprintf("/api/albums/%s/user/%s", album.ID, targetUserID),
					Method: "DELETE",
					// Explicitly set no body for DELETE request
				},
			},
			Revert: []plan.Request{
				{
					Path:   fmt.Sprintf("/api/albums/%s/users", album.ID),
					Method: "PUT",
					Body:   revertBodyJSON,
				},
			},
		})
	}

	return p, nil
}