package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/aayush/torrcli/internal/daemon"
	engineanacrolix "github.com/aayush/torrcli/internal/engine/anacrolix"
	"github.com/aayush/torrcli/internal/platform"
)

func main() {
	paths, err := platform.DefaultPaths()
	if err != nil {
		fatal(err)
	}
	torrentEngine, err := engineanacrolix.New()
	if err != nil {
		fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := daemon.New(paths, torrentEngine).Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "torrd:", err)
	os.Exit(1)
}
