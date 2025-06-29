package cmd

import (
	"github.com/spf13/cobra"
	"immich-manager/cmd/users"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Generate plans for Immich user operations",
}

func init() {
	planCmd.AddCommand(usersCmd)
	users.Init(usersCmd)
}
