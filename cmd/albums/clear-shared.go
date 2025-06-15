package albums

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"immich-manager/pkg/immich"
	"immich-manager/pkg/immich/albums/clearshared"
)

var ClearSharedCmd = &cobra.Command{
	Use:   "clear-shared [email]",
	Short: "Generate a plan to remove a user from all shared albums",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get environment variables
		immichServerUrl := os.Getenv("IMMICH_SERVER")
		if immichServerUrl == "" {
			return fmt.Errorf("IMMICH_SERVER environment variable is required")
		}

		immichApiKey := os.Getenv("IMMICH_TOKEN")
		if immichApiKey == "" {
			return fmt.Errorf("IMMICH_TOKEN environment variable is required")
		}

		// Parse arguments
		email := args[0]

		// Create client and generator
		client := immich.NewClient(immichServerUrl, immichApiKey)
		generator := clearshared.NewGenerator(client, email)

		// Generate plan
		plan, err := generator.Generate()
		if err != nil {
			return err
		}

		// Output plan as JSON
		planJSON, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return fmt.Errorf("marshaling plan to JSON: %w", err)
		}

		fmt.Println(string(planJSON))
		return nil
	},
}