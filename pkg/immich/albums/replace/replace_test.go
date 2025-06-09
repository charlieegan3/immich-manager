package replace

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"immich-manager/pkg/immich"
)

func TestGenerator_Generate(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/albums" {
			albums := []immich.Album{
				{ID: "1", Name: "foo album"},
				{ID: "2", Name: "bar album"},
				{ID: "3", Name: "foo bar album"},
			}
			json.NewEncoder(w).Encode(albums)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	// Create client and generator
	client := immich.NewClient(server.URL, "test-token")
	generator := NewGenerator(client, "foo", "baz")

	// Generate plan
	p, err := generator.Generate()
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Verify plan
	if len(p.Operations) != 2 {
		t.Errorf("Expected 2 operations, got %d", len(p.Operations))
	}

	// Verify first operation
	op1 := p.Operations[0]
	if op1.Apply.Path != "/api/albums/1" {
		t.Errorf("Expected path /api/albums/1, got %s", op1.Apply.Path)
	}
	if op1.Apply.Method != "PATCH" {
		t.Errorf("Expected method PATCH, got %s", op1.Apply.Method)
	}

	var body1 map[string]string
	if err := json.Unmarshal(op1.Apply.Body, &body1); err != nil {
		t.Fatalf("Failed to unmarshal body: %v", err)
	}
	if body1["albumName"] != "baz album" {
		t.Errorf("Expected albumName 'baz album', got %s", body1["albumName"])
	}

	// Verify revert operation
	if op1.Revert.Path != "/api/albums/1" {
		t.Errorf("Expected revert path /api/albums/1, got %s", op1.Revert.Path)
	}
	if op1.Revert.Method != "PATCH" {
		t.Errorf("Expected revert method PATCH, got %s", op1.Revert.Method)
	}

	var revertBody1 map[string]string
	if err := json.Unmarshal(op1.Revert.Body, &revertBody1); err != nil {
		t.Fatalf("Failed to unmarshal revert body: %v", err)
	}
	if revertBody1["albumName"] != "foo album" {
		t.Errorf("Expected revert albumName 'foo album', got %s", revertBody1["albumName"])
	}

	// Verify second operation
	op2 := p.Operations[1]
	if op2.Apply.Path != "/api/albums/3" {
		t.Errorf("Expected path /api/albums/3, got %s", op2.Apply.Path)
	}
	if op2.Apply.Method != "PATCH" {
		t.Errorf("Expected method PATCH, got %s", op2.Apply.Method)
	}

	var body2 map[string]string
	if err := json.Unmarshal(op2.Apply.Body, &body2); err != nil {
		t.Fatalf("Failed to unmarshal body: %v", err)
	}
	if body2["albumName"] != "baz bar album" {
		t.Errorf("Expected albumName 'baz bar album', got %s", body2["albumName"])
	}

	// Verify second revert operation
	if op2.Revert.Path != "/api/albums/3" {
		t.Errorf("Expected revert path /api/albums/3, got %s", op2.Revert.Path)
	}
	if op2.Revert.Method != "PATCH" {
		t.Errorf("Expected revert method PATCH, got %s", op2.Revert.Method)
	}

	var revertBody2 map[string]string
	if err := json.Unmarshal(op2.Revert.Body, &revertBody2); err != nil {
		t.Fatalf("Failed to unmarshal revert body: %v", err)
	}
	if revertBody2["albumName"] != "foo bar album" {
		t.Errorf("Expected revert albumName 'foo bar album', got %s", revertBody2["albumName"])
	}
}
