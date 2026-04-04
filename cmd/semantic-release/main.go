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
	os.Exit(run())
}

// run executes the CLI and returns the exit code. Extracting the logic here
// ensures defer stop() runs before os.Exit in main — os.Exit does not run
// deferred functions, so placing it in main() directly would skip the signal
// deregistration.
func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.NewRootCmd().ExecuteContext(ctx); err != nil {
		// ErrQuietExit means the command already printed its own error output.
		// Exit with code 1 without printing anything further to avoid duplication.
		if !errors.Is(err, cli.ErrQuietExit) {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}
	return 0
}
