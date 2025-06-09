package albums

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"immich-manager/pkg/immich"
	"immich-manager/pkg/immich/albums/replace"
)

var ReplaceCmd = &cobra.Command{
	Use:   "replace [before] [after]",
	Short: "Generate a plan to replace text in album names",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		before := args[0]
		after := args[1]

		token := os.Getenv("IMMICH_TOKEN")
		if token == "" {
			return fmt.Errorf("IMMICH_TOKEN environment variable is required")
		}

		server := os.Getenv("IMMICH_SERVER")
		if server == "" {
			return fmt.Errorf("IMMICH_SERVER environment variable is required")
		}

		client := immich.NewClient(server, token)
		generator := replace.NewGenerator(client, before, after)

		plan, err := generator.Generate()
		if err != nil {
			return fmt.Errorf("generating plan: %w", err)
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(plan); err != nil {
			return fmt.Errorf("encoding plan: %w", err)
		}

		return nil
	},
}
