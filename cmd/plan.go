package cmd

import (
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Generate plans for Immich operations",
}

func init() {
	rootCmd.AddCommand(planCmd)
}