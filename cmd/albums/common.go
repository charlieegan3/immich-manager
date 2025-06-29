package albums

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"immich-manager/pkg/immich"
)

// getClient returns a configured Immich client.
func getClient() (*immich.Client, error) {
	token := os.Getenv("IMMICH_TOKEN")
	if token == "" {
		return nil, errors.New("IMMICH_TOKEN environment variable is required")
	}

	server := os.Getenv("IMMICH_SERVER")
	if server == "" {
		return nil, errors.New("IMMICH_SERVER environment variable is required")
	}

	return immich.NewClient(server, token), nil
}

// outputPlan encodes and outputs a plan to stdout.
func outputPlan(plan any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(plan); err != nil {
		return fmt.Errorf("encoding plan: %w", err)
	}

	return nil
}
