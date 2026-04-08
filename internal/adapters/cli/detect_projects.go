package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func newDetectProjectsCmd(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "detect-projects",
		Short: "Discover projects in the repository",
		Long:  `Detect all projects/modules in the repository using configured discovery strategies.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDetectProjects(cmd, args, opts)
		},
	}
}

func runDetectProjects(cmd *cobra.Command, _ []string, opts *rootOptions) error {
	ctx := cmd.Context()

	container, workDir, err := buildContainerWithWorkDir(opts)
	if err != nil {
		return err
	}

	projects, err := container.ProjectDetector().Detect(ctx, workDir)
	if err != nil {
		return fmt.Errorf("detecting projects: %w", err)
	}

	if opts.jsonOut {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(projects); err != nil {
			return fmt.Errorf("encoding projects as JSON: %w", err)
		}
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Discovered %d project(s):\n\n", len(projects))
	for _, p := range projects {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", p.Name)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Path:   %s\n", p.Path)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Type:   %s\n", p.Type)
		if p.ModulePath != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Module: %s\n", p.ModulePath)
		}
		if p.TagPrefix != "" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    Tags:   %s*\n", p.TagPrefix)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}
