package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"immich-manager/pkg/immich"
	"immich-manager/pkg/immich/applier"
	"immich-manager/pkg/plan"
)

var revertDryRun bool

var revertCmd = &cobra.Command{
	Use:   "revert [plan-file]",
	Short: "Revert changes from a plan",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		planFile := args[0]

		token := os.Getenv("IMMICH_TOKEN")
		if token == "" {
			return errors.New("IMMICH_TOKEN environment variable is required")
		}

		server := os.Getenv("IMMICH_SERVER")
		if server == "" {
			return errors.New("IMMICH_SERVER environment variable is required")
		}

		p, err := plan.Load(planFile)
		if err != nil {
			return fmt.Errorf("loading plan: %w", err)
		}

		client := immich.NewClient(server, token)
		a := applier.NewApplier(client)

		opts := &applier.ApplyOptions{
			DryRun: revertDryRun,
			Writer: os.Stdout,
		}

		if err := a.Revert(p, opts); err != nil {
			return fmt.Errorf("reverting plan: %w", err)
		}

		if !revertDryRun {
			fmt.Fprintf(os.Stderr, "Successfully reverted plan with %d operations\n", len(p.Operations))
		}

		return nil
	},
}

func init() {
	revertCmd.Flags().BoolVar(&revertDryRun, "dry-run", false,
		"Print operations that would be performed without executing them")
	rootCmd.AddCommand(revertCmd)
}
