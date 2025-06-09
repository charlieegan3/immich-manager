package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"immich-manager/pkg/immich"
	"immich-manager/pkg/immich/applier"
	"immich-manager/pkg/plan"
)

var dryRun bool

var applyCmd = &cobra.Command{
	Use:   "apply [plan-file]",
	Short: "Apply a plan to the Immich API",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		planFile := args[0]

		token := os.Getenv("IMMICH_TOKEN")
		if token == "" {
			return fmt.Errorf("IMMICH_TOKEN environment variable is required")
		}

		server := os.Getenv("IMMICH_SERVER")
		if server == "" {
			return fmt.Errorf("IMMICH_SERVER environment variable is required")
		}

		p, err := plan.Load(planFile)
		if err != nil {
			return fmt.Errorf("loading plan: %w", err)
		}

		client := immich.NewClient(server, token)
		a := applier.NewApplier(client)

		opts := &applier.ApplyOptions{
			DryRun: dryRun,
			Writer: os.Stdout,
		}

		if err := a.Apply(p, opts); err != nil {
			return fmt.Errorf("applying plan: %w", err)
		}

		if !dryRun {
			fmt.Printf("Successfully applied plan with %d operations\n", len(p.Operations))
		}
		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print operations that would be performed without executing them")
	rootCmd.AddCommand(applyCmd)
}
