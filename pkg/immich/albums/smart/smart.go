package smart

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

// ErrAlbumNotFound is returned when a smart album does not exist.
var ErrAlbumNotFound = errors.New("album not found")

// Generator generates a plan for managing a smart album that aggregates assets from all shared albums.
type Generator struct {
	client *immich.Client
	email  string
}

// NewGenerator creates a new smart album plan generator.
func NewGenerator(client *immich.Client, email string) *Generator {
	return &Generator{
		client: client,
		email:  email,
	}
}

// Generate creates a plan for managing a smart album.
func (g *Generator) Generate() (*plan.Plan, error) {
	// 1. Find the user by email
	user, err := g.findUserByEmail()
	if err != nil {
		return nil, fmt.Errorf("finding user by email: %w", err)
	}

	// 2. Generate the "All NAME" album name
	smartAlbumName := "All " + user.Name

	// 3. Find the "All NAME" album
	smartAlbum, err := g.findSmartAlbum(smartAlbumName)
	if err != nil {
		if errors.Is(err, ErrAlbumNotFound) {
			return nil, fmt.Errorf("smart album '%s' does not exist - please create it first", smartAlbumName)
		}

		return nil, fmt.Errorf("finding smart album: %w", err)
	}

	// 4. Get all shared albums for the user
	sharedAlbums, err := g.getSharedAlbums(user.ID)
	if err != nil {
		return nil, fmt.Errorf("getting shared albums: %w", err)
	}

	// 5. Collect all assets from shared albums
	sharedAssets, err := g.getAssetsFromSharedAlbums(sharedAlbums)
	if err != nil {
		return nil, fmt.Errorf("getting assets from shared albums: %w", err)
	}

	// 6. Get current assets in the smart album
	smartAlbumAssets, err := g.getAlbumAssets(smartAlbum.ID)
	if err != nil {
		return nil, fmt.Errorf("getting assets in smart album: %w", err)
	}

	// 7. Create plan operations
	p := &plan.Plan{
		Operations: make([]plan.Operation, 0),
	}

	// Find assets to remove (in smart album but not in any shared album)
	assetsToRemove := make([]string, 0)

	for assetID := range smartAlbumAssets {
		if _, exists := sharedAssets[assetID]; !exists {
			assetsToRemove = append(assetsToRemove, assetID)
		}
	}

	// Create operations for removing assets
	if len(assetsToRemove) > 0 {
		removeBody := map[string]any{
			"ids": assetsToRemove,
		}

		removeBodyJSON, err := json.Marshal(removeBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling remove assets body: %w", err)
		}

		// Revert would be to add these assets back to the album
		addRemovedBody := map[string]any{
			"ids": assetsToRemove,
		}

		addRemovedBodyJSON, err := json.Marshal(addRemovedBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling add removed assets body: %w", err)
		}

		p.Operations = append(p.Operations, plan.Operation{
			Apply: []plan.Request{
				{
					Path:   fmt.Sprintf("/api/albums/%s/assets", smartAlbum.ID),
					Method: "DELETE",
					Body:   removeBodyJSON,
				},
			},
			Revert: []plan.Request{
				{
					Path:   fmt.Sprintf("/api/albums/%s/assets", smartAlbum.ID),
					Method: "PUT",
					Body:   addRemovedBodyJSON,
				},
			},
		})
	}

	// Find assets to add (in shared albums but not in smart album)
	assetsToAdd := make([]string, 0)

	for assetID := range sharedAssets {
		if _, exists := smartAlbumAssets[assetID]; !exists {
			assetsToAdd = append(assetsToAdd, assetID)
		}
	}

	// Create operations for adding assets
	if len(assetsToAdd) > 0 {
		addBody := map[string]any{
			"ids": assetsToAdd,
		}

		addBodyJSON, err := json.Marshal(addBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling add assets body: %w", err)
		}

		// Revert would be to remove these assets from the album
		removeAddedBody := map[string]any{
			"ids": assetsToAdd,
		}

		removeAddedBodyJSON, err := json.Marshal(removeAddedBody)
		if err != nil {
			return nil, fmt.Errorf("marshaling remove added assets body: %w", err)
		}

		p.Operations = append(p.Operations, plan.Operation{
			Apply: []plan.Request{
				{
					Path:   fmt.Sprintf("/api/albums/%s/assets", smartAlbum.ID),
					Method: "PUT",
					Body:   addBodyJSON,
				},
			},
			Revert: []plan.Request{
				{
					Path:   fmt.Sprintf("/api/albums/%s/assets", smartAlbum.ID),
					Method: "DELETE",
					Body:   removeAddedBodyJSON,
				},
			},
		})
	}

	// If no operations are needed, return an empty plan
	if len(p.Operations) == 0 {
		return p, nil
	}

	return p, nil
}

// findUserByEmail finds a user by their email address.
func (g *Generator) findUserByEmail() (*immich.User, error) {
	req, err := g.client.NewRequest("GET", "/api/users", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for users: %w", err)
	}

	var users []immich.User
	if err := g.client.Do(req, &users); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	for i, user := range users {
		if strings.EqualFold(user.Email, g.email) {
			return &users[i], nil
		}
	}

	return nil, fmt.Errorf("no user found with email '%s'", g.email)
}

// findSmartAlbum finds the "All NAME" album, or returns nil if it doesn't exist.
func (g *Generator) findSmartAlbum(albumName string) (*immich.Album, error) {
	// Try to find an existing album with this name
	req, err := g.client.NewRequest("GET", "/api/albums", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for albums: %w", err)
	}

	var albums []immich.Album
	if err := g.client.Do(req, &albums); err != nil {
		return nil, fmt.Errorf("getting albums: %w", err)
	}

	for _, album := range albums {
		if album.Name == albumName {
			return &album, nil
		}
	}

	// Album doesn't exist
	return nil, ErrAlbumNotFound
}

// getSharedAlbums gets all albums shared with the user.
func (g *Generator) getSharedAlbums(userID string) ([]immich.Album, error) {
	req, err := g.client.NewRequest("GET", "/api/albums", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for albums: %w", err)
	}

	var allAlbums []immich.Album
	if err := g.client.Do(req, &allAlbums); err != nil {
		return nil, fmt.Errorf("getting albums: %w", err)
	}

	// Filter to include only albums shared with the user
	sharedAlbums := make([]immich.Album, 0)

	for _, album := range allAlbums {
		// Skip the "All NAME" album itself to avoid recursion
		if strings.HasPrefix(album.Name, "All ") {
			continue
		}

		// Check if the user is in this album

		for _, albumUser := range album.AlbumUsers {
			if albumUser.User.ID == userID {
				sharedAlbums = append(sharedAlbums, album)

				break
			}
		}
	}

	return sharedAlbums, nil
}

// getAssetsFromSharedAlbums gets all unique assets from the shared albums.
func (g *Generator) getAssetsFromSharedAlbums(albums []immich.Album) (map[string]bool, error) {
	uniqueAssets := make(map[string]bool)

	for _, album := range albums {
		assets, err := g.getAlbumAssets(album.ID)
		if err != nil {
			return nil, fmt.Errorf("getting assets for album %s: %w", album.ID, err)
		}

		// Add all assets to the unique set
		for assetID := range assets {
			uniqueAssets[assetID] = true
		}
	}

	return uniqueAssets, nil
}

// getAlbumAssets gets all assets in an album.
func (g *Generator) getAlbumAssets(albumID string) (map[string]bool, error) {
	req, err := g.client.NewRequest("GET", fmt.Sprintf("/api/albums/%s?withoutAssets=false", albumID), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for album assets: %w", err)
	}

	var album struct {
		Assets []struct {
			ID string `json:"id"`
		} `json:"assets"`
	}

	if err := g.client.Do(req, &album); err != nil {
		return nil, fmt.Errorf("getting album with assets: %w", err)
	}

	assets := make(map[string]bool)
	for _, asset := range album.Assets {
		assets[asset.ID] = true
	}

	return assets, nil
}
