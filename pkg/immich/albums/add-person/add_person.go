// Package addperson provides functionality to add users to Immich albums based on person ID.
package addperson

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

// Generator generates a plan for adding a user to albums containing assets of a specific person.
type Generator struct {
	client   *immich.Client
	personID string
	email    string
}

// NewGenerator creates a new plan generator for adding a user to albums based on person ID.
func NewGenerator(client *immich.Client, personID, email string) *Generator {
	return &Generator{
		client:   client,
		personID: personID,
		email:    email,
	}
}

// SearchMetadataRequest represents the request body for searching metadata by person ID.
type SearchMetadataRequest struct {
	Page      string   `json:"page"`
	PersonIDs []string `json:"personIds"`
}

// Asset represents an asset in the search results.
type Asset struct {
	ID string `json:"id"`
}

// SearchMetadataResponse represents the response from the metadata search API.
type SearchMetadataResponse struct {
	Assets struct {
		Items    []Asset `json:"items"`
		NextPage *string `json:"nextPage"`
	} `json:"assets"`
}

// AlbumsByAssetResponse represents the response from getting albums by asset ID.
type AlbumsByAssetResponse []struct {
	ID string `json:"id"`
}

// Generate creates a plan for adding a user to albums containing assets of the specified person.
func (g *Generator) Generate() (*plan.Plan, error) {
	// Step 1: Get all asset IDs for the person (paginated)
	assetIDs, err := g.getAllAssetsForPerson()
	if err != nil {
		return nil, fmt.Errorf("getting assets for person: %w", err)
	}

	if len(assetIDs) == 0 {
		return nil, fmt.Errorf("no assets found for person ID '%s'", g.personID)
	}

	// Step 2: Get unique album IDs containing these assets
	albumIDs, err := g.getUniqueAlbumIDs(assetIDs)
	if err != nil {
		return nil, fmt.Errorf("getting album IDs: %w", err)
	}

	if len(albumIDs) == 0 {
		return nil, fmt.Errorf("no albums found containing assets for person ID '%s'", g.personID)
	}

	// Step 3: Get all users and find the one with matching email
	targetUser, err := g.findUserByEmail()
	if err != nil {
		return nil, err
	}

	// Step 4: Get album details and create operations
	return g.createPlanForAlbums(albumIDs, targetUser)
}

// getAllAssetsForPerson retrieves all asset IDs for a person using paginated search.
func (g *Generator) getAllAssetsForPerson() ([]string, error) {
	var allAssetIDs []string

	page := "1"

	for {
		searchRequest := SearchMetadataRequest{
			Page:      page,
			PersonIDs: []string{g.personID},
		}

		req, err := g.client.NewRequest("POST", "/api/search/metadata", searchRequest)
		if err != nil {
			return nil, fmt.Errorf("creating search metadata request: %w", err)
		}

		var response SearchMetadataResponse
		if err := g.client.Do(req, &response); err != nil {
			return nil, fmt.Errorf("searching metadata for person %s (page %s): %w", g.personID, page, err)
		}

		// Collect asset IDs from this page
		for _, asset := range response.Assets.Items {
			allAssetIDs = append(allAssetIDs, asset.ID)
		}

		// Check if there are more pages
		if response.Assets.NextPage == nil {
			break
		}

		page = *response.Assets.NextPage
	}

	return allAssetIDs, nil
}

// getUniqueAlbumIDs gets unique album IDs containing the specified assets.
func (g *Generator) getUniqueAlbumIDs(assetIDs []string) ([]string, error) {
	albumIDSet := make(map[string]bool)

	for _, assetID := range assetIDs {
		req, err := g.client.NewRequest("GET", "/api/albums?assetId="+assetID, map[string]interface{}{})
		if err != nil {
			return nil, fmt.Errorf("creating request for albums containing asset %s: %w", assetID, err)
		}

		var albums AlbumsByAssetResponse
		if err := g.client.Do(req, &albums); err != nil {
			return nil, fmt.Errorf("getting albums for asset %s: %w", assetID, err)
		}

		for _, album := range albums {
			albumIDSet[album.ID] = true
		}
	}

	// Convert set to slice
	albumIDs := make([]string, 0, len(albumIDSet))

	for albumID := range albumIDSet {
		albumIDs = append(albumIDs, albumID)
	}

	return albumIDs, nil
}

// findUserByEmail finds a user by email address.
func (g *Generator) findUserByEmail() (*immich.User, error) {
	req, err := g.client.NewRequest("GET", "/api/users", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for users: %w", err)
	}

	var users []immich.User
	if err := g.client.Do(req, &users); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	// Find user with matching email
	for i, user := range users {
		if strings.EqualFold(user.Email, g.email) {
			return &users[i], nil
		}
	}

	return nil, fmt.Errorf("no user found with email '%s'", g.email)
}

// createPlanForAlbums creates a plan for adding the user to the specified albums.
func (g *Generator) createPlanForAlbums(albumIDs []string, targetUser *immich.User) (*plan.Plan, error) {
	p := &plan.Plan{
		Operations: make([]plan.Operation, 0, len(albumIDs)),
	}

	for _, albumID := range albumIDs {
		// Get album details to check if user is already in the album
		req, err := g.client.NewRequest("GET", "/api/albums/"+albumID, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request for album %s: %w", albumID, err)
		}

		var album immich.Album
		if err := g.client.Do(req, &album); err != nil {
			return nil, fmt.Errorf("getting album %s: %w", albumID, err)
		}

		// Check if user is already in the album
		userAlreadyInAlbum := false

		for _, albumUser := range album.AlbumUsers {
			if albumUser.User.ID == targetUser.ID {
				userAlreadyInAlbum = true

				break
			}
		}

		if userAlreadyInAlbum {
			continue
		}

		// Add user to album request
		addUserBody := map[string]any{
			"albumUsers": []map[string]string{
				{
					"role":   "viewer",
					"userId": targetUser.ID,
				},
			},
		}

		addUserBodyJSON, err := json.Marshal(addUserBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling add user body: %w", err)
		}

		// Remove user from album request
		removeUserPath := fmt.Sprintf("/api/albums/%s/user/%s", album.ID, targetUser.ID)

		p.Operations = append(p.Operations, plan.Operation{
			Apply: []plan.Request{
				{
					Path:   fmt.Sprintf("/api/albums/%s/users", album.ID),
					Method: http.MethodPut,
					Body:   addUserBodyJSON,
				},
			},
			Revert: []plan.Request{
				{
					Path:   removeUserPath,
					Method: http.MethodDelete,
					Body:   nil,
				},
			},
		})
	}

	if len(p.Operations) == 0 {
		return nil, errors.New("no changes needed - user is already in all albums containing assets for this person")
	}

	return p, nil
}
