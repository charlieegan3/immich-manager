package cmd

import (
	"github.com/spf13/cobra"
	"immich-manager/cmd/albums"
)

var albumsCmd = &cobra.Command{
	Use:   "albums",
	Short: "Manage Immich albums",
}

func init() {
	planCmd.AddCommand(albumsCmd)
	albumsCmd.AddCommand(albums.ReplaceCmd)
	albumsCmd.AddCommand(albums.AddUserCmd)
	albumsCmd.AddCommand(albums.ClearSharedCmd)
}
