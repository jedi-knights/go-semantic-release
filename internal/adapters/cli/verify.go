package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVerifyCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "verify",
		Short: "Verify release conditions are met",
		Long:  `Check that all prerequisites for a release are satisfied (branch policy, GitHub config, etc.).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVerify(cmd, args, opts)
		},
	}
}

func runVerify(cmd *cobra.Command, _ []string, opts *rootOptions) error {
	ctx := cmd.Context()

	container, _, err := buildContainerWithWorkDir(opts)
	if err != nil {
		return err
	}

	result, err := container.ConditionVerifier().Verify(ctx)
	if err != nil {
		return fmt.Errorf("verifying conditions: %w", err)
	}

	if result.Passed {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "All release conditions verified.")
		return nil
	}

	// Failures already printed to stderr; return ErrQuietExit so main exits
	// with code 1 without printing a redundant error message.
	_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "Release conditions not met:")
	for _, f := range result.Failures {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "  [FAIL] %s\n", f)
	}
	return ErrQuietExit
}
