package albums

import (
	"fmt"

	"github.com/spf13/cobra"
	"immich-manager/pkg/immich/albums/replace"
)

var ReplaceCmd = &cobra.Command{
	Use:   "replace [before] [after]",
	Short: "Generate a plan to replace text in album names",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		before := args[0]
		after := args[1]

		client, err := getClient()
		if err != nil {
			return err
		}

		generator := replace.NewGenerator(client, before, after)

		plan, err := generator.Generate()
		if err != nil {
			return fmt.Errorf("generating plan: %w", err)
		}

		return outputPlan(plan)
	},
}
