package replace

import (
	"encoding/json"
	"fmt"
	"strings"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/plan"
)

// Generator generates a plan for renaming albums.
type Generator struct {
	client *immich.Client
	before string
	after  string
}

// NewGenerator creates a new rename plan generator.
func NewGenerator(client *immich.Client, before, after string) *Generator {
	return &Generator{
		client: client,
		before: before,
		after:  after,
	}
}

// Generate creates a plan for renaming albums.
func (g *Generator) Generate() (*plan.Plan, error) {
	// Get all albums
	req, err := g.client.NewRequest("GET", "/api/albums", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	var albums []immich.Album
	if err := g.client.Do(req, &albums); err != nil {
		return nil, fmt.Errorf("getting albums: %w", err)
	}

	p := &plan.Plan{
		Operations: make([]plan.Operation, 0),
	}

	for _, album := range albums {
		if !strings.Contains(album.Name, g.before) {
			continue
		}

		newName := strings.ReplaceAll(album.Name, g.before, g.after)
		// Skip if the name hasn't changed
		if newName == album.Name {
			continue
		}

		// Create update operation
		updateBody := map[string]string{
			"albumName": newName,
		}
		jsonBody, err := json.Marshal(updateBody)

		if err != nil {
			return nil, fmt.Errorf("marshaling update body: %w", err)
		}

		// Create revert operation
		revertBody := map[string]string{
			"albumName": album.Name,
		}
		revertJSONBody, err := json.Marshal(revertBody)

		if err != nil {
			return nil, fmt.Errorf("marshaling revert body: %w", err)
		}

		p.Operations = append(p.Operations, plan.Operation{
			Apply: []plan.Request{
				{
					Path:   "/api/albums/" + album.ID,
					Method: "PATCH",
					Body:   jsonBody,
				},
			},
			Revert: []plan.Request{
				{
					Path:   "/api/albums/" + album.ID,
					Method: "PATCH",
					Body:   revertJSONBody,
				},
			},
		})
	}

	return p, nil
}
