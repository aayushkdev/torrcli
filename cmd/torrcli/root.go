package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aayush/torrcli/internal/client"
	"github.com/aayush/torrcli/internal/platform"
	"github.com/aayush/torrcli/internal/tui"
	"github.com/spf13/cobra"
)

type application struct {
	daemon *client.Client
}

var app application

var rootCmd = &cobra.Command{
	Use:               "torrcli",
	Short:             "A terminal BitTorrent client",
	SilenceUsage:      true,
	SilenceErrors:     true,
	PersistentPreRunE: ensureDaemon,
	RunE: func(command *cobra.Command, _ []string) error {
		return runTUI(command.Context())
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(addCmd, listCmd, pauseCmd, resumeCmd, removeCmd, priorityCmd, statusCmd)
}

func ensureDaemon(command *cobra.Command, _ []string) error {
	if app.daemon != nil {
		return nil
	}
	paths, err := platform.DefaultPaths()
	if err != nil {
		return err
	}
	deadline, cancel := context.WithTimeout(command.Context(), time.Second)
	defer cancel()

	daemon := client.New(paths.SocketPath)
	if _, err := daemon.Ping(deadline); err != nil {
		return fmt.Errorf("torrd is unavailable")
	}
	app.daemon = daemon
	return nil
}

func runTUI(context.Context) error {
	return tui.Run()
}
