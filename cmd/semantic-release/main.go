package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/cli"
)

func main() {
	// Wire OS interrupt signals into the root context so long-running git and
	// network calls respect Ctrl-C. Deferring stop() here guarantees the
	// registration is always released, even when a command returns an error.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.NewRootCmd().ExecuteContext(ctx); err != nil {
		// ErrQuietExit means the command already printed its own error output.
		// Exit with code 1 without printing anything further to avoid duplication.
		if !errors.Is(err, cli.ErrQuietExit) {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}
