package plan

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestPlan_Save_Load(t *testing.T) {
	// Create test plan
	p := &Plan{
		Operations: []Operation{
			{
				Apply: Request{
					Path:   "/api/albums/1",
					Method: "PATCH",
					Body:   json.RawMessage(`{"albumName": "new name"}`),
				},
				Revert: Request{
					Path:   "/api/albums/1",
					Method: "PATCH",
					Body:   json.RawMessage(`{"albumName": "old name"}`),
				},
			},
		},
	}

	// Save plan to buffer
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(p); err != nil {
		t.Fatalf("Failed to encode plan: %v", err)
	}

	// Load plan from buffer
	loaded, err := LoadFromReader(&buf)
	if err != nil {
		t.Fatalf("Failed to load plan: %v", err)
	}

	// Verify loaded plan
	if len(loaded.Operations) != 1 {
		t.Errorf("Expected 1 operation, got %d", len(loaded.Operations))
	}

	op := loaded.Operations[0]

	// Verify apply operation
	if op.Apply.Path != "/api/albums/1" {
		t.Errorf("Expected apply path /api/albums/1, got %s", op.Apply.Path)
	}
	if op.Apply.Method != "PATCH" {
		t.Errorf("Expected apply method PATCH, got %s", op.Apply.Method)
	}

	var applyBody map[string]string
	if err := json.Unmarshal(op.Apply.Body, &applyBody); err != nil {
		t.Fatalf("Failed to unmarshal apply body: %v", err)
	}
	if applyBody["albumName"] != "new name" {
		t.Errorf("Expected apply albumName 'new name', got %s", applyBody["albumName"])
	}

	// Verify revert operation
	if op.Revert.Path != "/api/albums/1" {
		t.Errorf("Expected revert path /api/albums/1, got %s", op.Revert.Path)
	}
	if op.Revert.Method != "PATCH" {
		t.Errorf("Expected revert method PATCH, got %s", op.Revert.Method)
	}

	var revertBody map[string]string
	if err := json.Unmarshal(op.Revert.Body, &revertBody); err != nil {
		t.Fatalf("Failed to unmarshal revert body: %v", err)
	}
	if revertBody["albumName"] != "old name" {
		t.Errorf("Expected revert albumName 'old name', got %s", revertBody["albumName"])
	}
}
