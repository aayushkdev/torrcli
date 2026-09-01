package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Args:  cobra.NoArgs,
	RunE: func(command *cobra.Command, _ []string) error {
		return showStatus(command.Context())
	},
}

func showStatus(ctx context.Context) error {
	info, err := app.daemon.Info(ctx)
	if err != nil {
		return err
	}
	fmt.Printf("torrd %s\n", info.DaemonVersion)
	fmt.Printf("protocol: %s\n", info.ProtocolVersion)
	fmt.Printf("uptime: %s\n", time.Since(info.StartedAt).Truncate(time.Second))
	fmt.Printf("socket: %s\n", info.SocketPath)
	return nil
}
