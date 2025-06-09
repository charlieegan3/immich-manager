package plan

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Operation represents a single API operation to be performed
type Operation struct {
	Apply  Request `json:"apply"`
	Revert Request `json:"revert"`
}

// Request represents a single API request
type Request struct {
	Path   string          `json:"path"`
	Method string          `json:"method"`
	Body   json.RawMessage `json:"body,omitempty"`
}

// Plan represents a series of operations to be performed
type Plan struct {
	Operations []Operation `json:"operations"`
}

// Generator is an interface for types that can generate plans
type Generator interface {
	Generate() (*Plan, error)
}

// Applier is an interface for types that can apply plans
type Applier interface {
	Apply(plan *Plan) error
}

// Save writes a plan to a file
func (p *Plan) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating plan file: %w", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(p); err != nil {
		return fmt.Errorf("encoding plan: %w", err)
	}

	return nil
}

// Load reads a plan from a file
func Load(path string) (*Plan, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening plan file: %w", err)
	}
	defer f.Close()

	return LoadFromReader(f)
}

// LoadFromReader reads a plan from an io.Reader
func LoadFromReader(r io.Reader) (*Plan, error) {
	var plan Plan
	if err := json.NewDecoder(r).Decode(&plan); err != nil {
		return nil, fmt.Errorf("decoding plan: %w", err)
	}
	return &plan, nil
}
