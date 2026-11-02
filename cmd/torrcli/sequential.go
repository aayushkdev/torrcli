package main

import (
	"fmt"
	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/spf13/cobra"
)

var sequentialCmd = &cobra.Command{
	Use:   "sequential ID on|off",
	Short: "Prioritize downloading a torrent from the beginning",
	Args:  cobra.ExactArgs(2),
	RunE: func(command *cobra.Command, args []string) error {
		if args[1] != "on" && args[1] != "off" {
			return fmt.Errorf("sequential mode must be on or off")
		}
		_, err := app.daemon.SetSequential(command.Context(), rpc.SetSequentialParams{ID: model.TorrentID(args[0]), Enabled: args[1] == "on"})
		return err
	},
}
