// Command torrcli is the terminal UI and command-line client for torrd.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aayush/torrcli/internal/client"
	"github.com/aayush/torrcli/internal/platform"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "status" {
		fmt.Fprintln(os.Stderr, "usage: torrcli status")
		os.Exit(2)
	}

	paths, err := platform.DefaultPaths()
	if err != nil {
		fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	daemon := client.New(paths.SocketPath)
	if _, err := daemon.Ping(ctx); err != nil {
		fatal(err)
	}
	info, err := daemon.Info(ctx)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("torrd %s\n", info.DaemonVersion)
	fmt.Printf("protocol: %s\n", info.ProtocolVersion)
	fmt.Printf("uptime: %s\n", time.Since(info.StartedAt).Truncate(time.Second))
	fmt.Printf("socket: %s\n", info.SocketPath)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "torrcli:", err)
	os.Exit(1)
}
