package main

import (
	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/spf13/cobra"
)

var resumeCmd = &cobra.Command{Use: "resume ID", Short: "Resume a torrent", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
	_, err := app.daemon.Resume(command.Context(), rpc.TorrentParams{ID: model.TorrentID(args[0])})
	return err
}}
