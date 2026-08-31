// Command torrd is the long-running BitTorrent daemon for torrcli.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aayush/torrcli/internal/daemon"
	"github.com/aayush/torrcli/internal/platform"
)

func main() {
	paths, err := platform.DefaultPaths()
	if err != nil {
		fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := daemon.New(paths).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "torrd:", err)
	os.Exit(1)
}
