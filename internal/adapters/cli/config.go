package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	adapterconfig "github.com/jedi-knights/go-semantic-release/internal/adapters/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
	}

	cmd.AddCommand(newConfigInitCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize a default configuration file",
		RunE:  runConfigInit,
	}
}

func runConfigInit(cmd *cobra.Command, _ []string) error {
	path := ".gosemrel.yaml"
	if err := adapterconfig.WriteDefaultConfig(path); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	fmt.Printf("Created %s with default configuration.\n", path)
	return nil
}
