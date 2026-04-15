// Package config handles loading and parsing of the go-filemover configuration.
package config

import (
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// TaskConfig represents the configuration for a single file watching task.
type TaskConfig struct {
	// Wd is the watched directory path.
	Wd string `toml:"wd"`
	// Td is the target directory path.
	Td string `toml:"td"`
	// Fg is a comma-separated list of glob patterns to match files against.
	Fg string `toml:"fg"`
}

// Config is a map of task IDs to their respective [TaskConfig].
type Config map[string]TaskConfig

// LoadConfig reads and parses the TOML configuration file from the given path.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ExpandPaths expands environment variables within the Wd and Td fields of the [TaskConfig].
func (t *TaskConfig) ExpandPaths() {
	t.Wd = os.ExpandEnv(t.Wd)
	t.Td = os.ExpandEnv(t.Td)
}

// GetPatterns splits the comma-separated Fg string into a slice of trimmed glob patterns.
// Empty patterns are filtered out.
func (t *TaskConfig) GetPatterns() []string {
	parts := strings.Split(t.Fg, ",")
	patterns := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			patterns = append(patterns, trimmed)
		}
	}
	return patterns
}
