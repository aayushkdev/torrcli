package main

import (
	"strconv"

	"github.com/aayush/torrcli/internal/model"
	"github.com/aayush/torrcli/internal/rpc"
	"github.com/spf13/cobra"
)

var priorityCmd = &cobra.Command{Use: "priority ID FILE_INDEX PRIORITY", Short: "Set a torrent file priority", Args: cobra.ExactArgs(3), RunE: func(command *cobra.Command, args []string) error {
	index, err := strconv.Atoi(args[1])
	if err != nil {
		return err
	}
	_, err = app.daemon.SetFilePriority(command.Context(), rpc.SetFilePriorityParams{ID: model.TorrentID(args[0]), FileIndex: index, Priority: model.FilePriority(args[2])})
	return err
}}
