// Package albums provides album management commands.
package albums

import (
	"fmt"

	"github.com/spf13/cobra"
	adduser "immich-manager/pkg/immich/albums/add-user"
)

var AddUserCmd = &cobra.Command{
	Use:   "add-user [search-term] [email]",
	Short: "Generate a plan to add a user to albums matching a search term",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		searchTerm := args[0]
		email := args[1]

		client, err := getClient()
		if err != nil {
			return err
		}

		generator := adduser.NewGenerator(client, searchTerm, email)

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
