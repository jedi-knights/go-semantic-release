package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newVerifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify release conditions are met",
		Long:  `Check that all prerequisites for a release are satisfied (branch policy, GitHub config, etc.).`,
		RunE:  runVerify,
	}
}

func runVerify(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	container, err := buildContainer()
	if err != nil {
		return err
	}

	result, err := container.ConditionVerifier().Verify(ctx)
	if err != nil {
		return err
	}

	if result.Passed {
		fmt.Println("All release conditions verified.")
		return nil
	}

	fmt.Fprintln(os.Stderr, "Release conditions not met:")
	for _, f := range result.Failures {
		fmt.Fprintf(os.Stderr, "  ✗ %s\n", f)
	}
	os.Exit(1)
	return nil
}
