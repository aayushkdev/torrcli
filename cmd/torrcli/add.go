package main

import (
	"fmt"

	"github.com/aayush/torrcli/internal/rpc"
	"github.com/spf13/cobra"
)

var addSavePath string
var addSequential, addPaused bool

var addCmd = &cobra.Command{
	Use:   "add SOURCE",
	Short: "Add a torrent",
	Args:  cobra.ExactArgs(1),
	RunE: func(command *cobra.Command, args []string) error {
		result, err := app.daemon.Add(command.Context(), rpc.AddTorrentParams{Source: args[0], SavePath: addSavePath, Sequential: addSequential, Paused: addPaused})
		if err != nil {
			return err
		}
		fmt.Println(result.Torrent.ID)
		return nil
	},
}

func init() {
	addCmd.Flags().BoolVar(&addSequential, "sequential", false, "Prioritize downloading from the beginning")
	addCmd.Flags().BoolVar(&addPaused, "paused", false, "Add without starting data transfer")
	addCmd.Flags().StringVar(&addSavePath, "save-path", "", "Directory for downloaded files")
}
