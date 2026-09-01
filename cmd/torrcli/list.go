package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List torrents",
	Args:  cobra.NoArgs,
	RunE: func(command *cobra.Command, _ []string) error {
		result, err := app.daemon.List(command.Context())
		if err != nil {
			return err
		}
		for _, torrent := range result.Torrents {
			fmt.Printf("%s\t%s\t%.1f%%\t↓%d B/s\t↑%d B/s\t%d peers (%d seeders, %d leechers)\t%s\n", torrent.ID, torrent.State, torrent.Progress*100, torrent.DownloadRate, torrent.UploadRate, torrent.ConnectedPeers, torrent.Seeders, torrent.Leechers, torrent.Name)
		}
		return nil
	},
}
