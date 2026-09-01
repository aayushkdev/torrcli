package main

import (
	"fmt"

	"github.com/aayush/torrcli/internal/rpc"
	"github.com/spf13/cobra"
)

var addSavePath string

var addCmd = &cobra.Command{
	Use:   "add SOURCE",
	Short: "Add a torrent",
	Args:  cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		result, err := app.daemon.Add(command.Context(), rpc.AddTorrentParams{Source: args[0], SavePath: addSavePath})
		if err != nil {
			return err
		}
		fmt.Println(result.Torrent.ID)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addSavePath, "save-path", "", "Directory for downloaded files")
}
