package plugins

import (
	"context"

	"github.com/jedi-knights/go-semantic-release/internal/domain"
	"github.com/jedi-knights/go-semantic-release/internal/ports"
)

// Compile-time interface compliance checks.
var (
	_ ports.Plugin               = (*CommitAnalyzerPlugin)(nil)
	_ ports.AnalyzeCommitsPlugin = (*CommitAnalyzerPlugin)(nil)
)

// CommitAnalyzerPlugin implements AnalyzeCommitsPlugin using conventional commits.
type CommitAnalyzerPlugin struct {
	parser      ports.CommitParser
	typeMapping map[string]domain.ReleaseType
}

// NewCommitAnalyzerPlugin creates the default commit analyzer plugin.
func NewCommitAnalyzerPlugin(parser ports.CommitParser, typeMapping map[string]domain.ReleaseType) *CommitAnalyzerPlugin {
	return &CommitAnalyzerPlugin{parser: parser, typeMapping: typeMapping}
}

func (p *CommitAnalyzerPlugin) Name() string { return "commit-analyzer" }

func (p *CommitAnalyzerPlugin) AnalyzeCommits(_ context.Context, rc *domain.ReleaseContext) (domain.ReleaseType, error) {
	highest := domain.ReleaseNone
	for i := range rc.Commits {
		rt := rc.Commits[i].ReleaseType(p.typeMapping)
		highest = highest.Higher(rt)
	}
	return highest, nil
}
