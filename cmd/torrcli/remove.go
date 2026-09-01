package main

import (
	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/spf13/cobra"
)

var removeData bool

var removeCmd = &cobra.Command{Use: "remove ID", Short: "Remove a torrent", Args: cobra.ExactArgs(1), RunE: func(command *cobra.Command, args []string) error {
	return app.daemon.Remove(command.Context(), rpc.RemoveTorrentParams{ID: model.TorrentID(args[0]), DeleteData: removeData})
}}

func init() {
	removeCmd.Flags().BoolVar(&removeData, "delete-data", false, "Delete downloaded torrent data")
}
