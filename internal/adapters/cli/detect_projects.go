package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newDetectProjectsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "detect-projects",
		Short: "Discover projects in the repository",
		Long:  `Detect all projects/modules in the repository using configured discovery strategies.`,
		RunE:  runDetectProjects,
	}
}

func runDetectProjects(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	container, err := buildContainer()
	if err != nil {
		return err
	}

	projects, err := container.ProjectDetector().Detect(ctx, getWorkDir())
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(projects)
	}

	fmt.Printf("Discovered %d project(s):\n\n", len(projects))
	for _, p := range projects {
		fmt.Printf("  %s\n", p.Name)
		fmt.Printf("    Path:   %s\n", p.Path)
		fmt.Printf("    Type:   %s\n", p.Type)
		if p.ModulePath != "" {
			fmt.Printf("    Module: %s\n", p.ModulePath)
		}
		if p.TagPrefix != "" {
			fmt.Printf("    Tags:   %s*\n", p.TagPrefix)
		}
		fmt.Println()
	}
	return nil
}
