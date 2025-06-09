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
	
	// Verify apply operations
	if len(op1.Apply) != 1 {
		t.Errorf("Expected 1 apply request, got %d", len(op1.Apply))
	} else {
		applyReq := op1.Apply[0]
		if applyReq.Path != "/api/albums/1" {
			t.Errorf("Expected path /api/albums/1, got %s", applyReq.Path)
		}
		if applyReq.Method != "PATCH" {
			t.Errorf("Expected method PATCH, got %s", applyReq.Method)
		}

		var body1 map[string]string
		if err := json.Unmarshal(applyReq.Body, &body1); err != nil {
			t.Fatalf("Failed to unmarshal body: %v", err)
		}
		if body1["albumName"] != "baz album" {
			t.Errorf("Expected albumName 'baz album', got %s", body1["albumName"])
		}
	}

	// Verify revert operations
	if len(op1.Revert) != 1 {
		t.Errorf("Expected 1 revert request, got %d", len(op1.Revert))
	} else {
		revertReq := op1.Revert[0]
		if revertReq.Path != "/api/albums/1" {
			t.Errorf("Expected revert path /api/albums/1, got %s", revertReq.Path)
		}
		if revertReq.Method != "PATCH" {
			t.Errorf("Expected revert method PATCH, got %s", revertReq.Method)
		}

		var revertBody1 map[string]string
		if err := json.Unmarshal(revertReq.Body, &revertBody1); err != nil {
			t.Fatalf("Failed to unmarshal revert body: %v", err)
		}
		if revertBody1["albumName"] != "foo album" {
			t.Errorf("Expected revert albumName 'foo album', got %s", revertBody1["albumName"])
		}
	}

	// Verify second operation
	op2 := p.Operations[1]
	
	// Verify apply operations
	if len(op2.Apply) != 1 {
		t.Errorf("Expected 1 apply request, got %d", len(op2.Apply))
	} else {
		applyReq := op2.Apply[0]
		if applyReq.Path != "/api/albums/3" {
			t.Errorf("Expected path /api/albums/3, got %s", applyReq.Path)
		}
		if applyReq.Method != "PATCH" {
			t.Errorf("Expected method PATCH, got %s", applyReq.Method)
		}

		var body2 map[string]string
		if err := json.Unmarshal(applyReq.Body, &body2); err != nil {
			t.Fatalf("Failed to unmarshal body: %v", err)
		}
		if body2["albumName"] != "baz bar album" {
			t.Errorf("Expected albumName 'baz bar album', got %s", body2["albumName"])
		}
	}

	// Verify revert operations
	if len(op2.Revert) != 1 {
		t.Errorf("Expected 1 revert request, got %d", len(op2.Revert))
	} else {
		revertReq := op2.Revert[0]
		if revertReq.Path != "/api/albums/3" {
			t.Errorf("Expected revert path /api/albums/3, got %s", revertReq.Path)
		}
		if revertReq.Method != "PATCH" {
			t.Errorf("Expected revert method PATCH, got %s", revertReq.Method)
		}

		var revertBody2 map[string]string
		if err := json.Unmarshal(revertReq.Body, &revertBody2); err != nil {
			t.Fatalf("Failed to unmarshal revert body: %v", err)
		}
		if revertBody2["albumName"] != "foo bar album" {
			t.Errorf("Expected revert albumName 'foo bar album', got %s", revertBody2["albumName"])
		}
	}
}
