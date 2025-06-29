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

var dryRun bool

var applyCmd = &cobra.Command{
	Use:   "apply [plan-file]",
	Short: "Apply a plan to the Immich API (use '-' or omit to read from stdin)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		token := os.Getenv("IMMICH_TOKEN")
		if token == "" {
			return errors.New("IMMICH_TOKEN environment variable is required")
		}

		server := os.Getenv("IMMICH_SERVER")
		if server == "" {
			return errors.New("IMMICH_SERVER environment variable is required")
		}

		var p *plan.Plan
		var err error

		if len(args) == 0 || args[0] == "-" {
			// Read from stdin
			p, err = plan.LoadFromReader(os.Stdin)
			if err != nil {
				return fmt.Errorf("loading plan from stdin: %w", err)
			}
		} else {
			// Read from file
			planFile := args[0]
			p, err = plan.Load(planFile)
			if err != nil {
				return fmt.Errorf("loading plan: %w", err)
			}
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
			fmt.Fprintf(os.Stderr, "Successfully applied plan with %d operations\n", len(p.Operations))
		}

		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print operations that would be performed without executing them")
	rootCmd.AddCommand(applyCmd)
}
