package albums

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"immich-manager/pkg/immich"
	adduser "immich-manager/pkg/immich/albums/add-user"
)

var AddUserCmd = &cobra.Command{
	Use:   "add-user [search-term] [email]",
	Short: "Generate a plan to add a user to albums matching a search term",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		searchTerm := args[0]
		email := args[1]

		token := os.Getenv("IMMICH_TOKEN")
		if token == "" {
			return fmt.Errorf("IMMICH_TOKEN environment variable is required")
		}

		server := os.Getenv("IMMICH_SERVER")
		if server == "" {
			return fmt.Errorf("IMMICH_SERVER environment variable is required")
		}

		client := immich.NewClient(server, token)
		generator := adduser.NewGenerator(client, searchTerm, email)

		p, err := generator.Generate()
		if err != nil {
			return fmt.Errorf("generating plan: %w", err)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(p); err != nil {
			return fmt.Errorf("encoding plan: %w", err)
		}

		return nil
	},
}


func init() {
	// This function is called by cmd/albums.go
}