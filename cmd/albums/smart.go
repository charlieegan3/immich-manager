package albums

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"immich-manager/pkg/immich"
	"immich-manager/pkg/immich/albums/smart"
)

var SmartCmd = &cobra.Command{
	Use:   "smart [email]",
	Short: "Generate a plan to create/maintain a smart album with all assets from albums shared with a user",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email := args[0]

		token := os.Getenv("IMMICH_TOKEN")
		if token == "" {
			return fmt.Errorf("IMMICH_TOKEN environment variable is required")
		}

		server := os.Getenv("IMMICH_SERVER")
		if server == "" {
			return fmt.Errorf("IMMICH_SERVER environment variable is required")
		}

		client := immich.NewClient(server, token)
		generator := smart.NewGenerator(client, email)

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