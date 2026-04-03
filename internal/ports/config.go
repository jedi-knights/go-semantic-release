package ports

import "github.com/jedi-knights/go-semantic-release/internal/domain"

// ConfigProvider loads and exposes the release configuration.
type ConfigProvider interface {
	Load(path string) (domain.Config, error)
}
