package main

import (
	"fmt"
	"os"

	"github.com/jedi-knights/go-semantic-release/internal/adapters/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
