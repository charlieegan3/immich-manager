package adduser

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

// Generator generates a plan for adding a user to albums matching a search term.
type Generator struct {
	client     *immich.Client
	searchTerm string
	email      string
}

// NewGenerator creates a new plan generator for adding a user to albums.
func NewGenerator(client *immich.Client, searchTerm, email string) *Generator {
	return &Generator{
		client:     client,
		searchTerm: searchTerm,
		email:      email,
	}
}

// Generate creates a plan for adding a user to albums matching the search term.
func (g *Generator) Generate() (*plan.Plan, error) {
	// Get all albums
	req, err := g.client.NewRequest("GET", "/api/albums", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for albums: %w", err)
	}

	var albums []immich.Album
	if err := g.client.Do(req, &albums); err != nil {
		return nil, fmt.Errorf("getting albums: %w", err)
	}

	// Filter albums by search term
	filteredAlbums := make([]immich.Album, 0)

	for _, album := range albums {
		if searchContains(album.Name, g.searchTerm) {
			filteredAlbums = append(filteredAlbums, album)
		}
	}

	if len(filteredAlbums) == 0 {
		return nil, fmt.Errorf("no albums found matching search term '%s'", g.searchTerm)
	}

	// Get all users and find the one with matching email
	req, err = g.client.NewRequest("GET", "/api/users", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request for users: %w", err)
	}

	var users []immich.User
	if err := g.client.Do(req, &users); err != nil {
		return nil, fmt.Errorf("getting users: %w", err)
	}

	// Find user with matching email
	var targetUser *immich.User

	for i, user := range users {
		if strings.EqualFold(user.Email, g.email) {
			targetUser = &users[i]

			break
		}
	}

	if targetUser == nil {
		return nil, fmt.Errorf("no user found with email '%s'", g.email)
	}

	p := &plan.Plan{
		Operations: make([]plan.Operation, 0, len(filteredAlbums)),
	}

	// Create operations for each album
	for _, album := range filteredAlbums {
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
					Method: "PUT",
					Body:   addUserBodyJSON,
				},
			},
			Revert: []plan.Request{
				{
					Path:   removeUserPath,
					Method: "DELETE",
					Body:   nil,
				},
			},
		})
	}

	if len(p.Operations) == 0 {
		return nil, errors.New("no changes needed - user is already in all matching albums")
	}

	return p, nil
}

// searchContains checks if the target string contains the search term
// (case-insensitive substring search).
func searchContains(target, search string) bool {
	targetLower := strings.ToLower(target)
	searchLower := strings.ToLower(search)

	return strings.Contains(targetLower, searchLower)
}
