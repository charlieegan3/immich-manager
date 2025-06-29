// Package albums provides album management commands.
package albums

import (
	"fmt"

	"github.com/spf13/cobra"
	addperson "immich-manager/pkg/immich/albums/add-person"
)

var AddPersonCmd = &cobra.Command{
	Use:   "add-person [person-id] [email]",
	Short: "Generate a plan to add a user to albums containing assets of a specific person",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		personID := args[0]
		email := args[1]

		client, err := getClient()
		if err != nil {
			return err
		}

		generator := addperson.NewGenerator(client, personID, email)

		plan, err := generator.Generate()
		if err != nil {
			return fmt.Errorf("generating plan: %w", err)
		}

		return outputPlan(plan)
	},
}

func init() {
	// This function is called by cmd/albums.go
}
