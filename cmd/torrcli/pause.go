package main

import (
	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/spf13/cobra"
)

var pauseCmd = &cobra.Command{Use: "pause ID", Short: "Pause a torrent", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
	_, err := app.daemon.Pause(command.Context(), rpc.TorrentParams{ID: model.TorrentID(args[0])})
	return err
}}
