package albums

import (
	"fmt"

	"github.com/spf13/cobra"
	"immich-manager/pkg/immich/albums/clearshared"
)

var ClearSharedCmd = &cobra.Command{
	Use:   "clear-shared [email]",
	Short: "Generate a plan to remove a user from all shared albums",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		email := args[0]

		client, err := getClient()
		if err != nil {
			return err
		}

		generator := clearshared.NewGenerator(client, email)

		plan, err := generator.Generate()
		if err != nil {
			return fmt.Errorf("generating plan: %w", err)
		}

		return outputPlan(plan)
	},
}
