// Package config handles CLI configuration and content root discovery.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultContentDir = "content"

// Config holds resolved runtime configuration.
type Config struct {
	ContentRoot string
}

// Load resolves configuration, preferring the explicit override, then the
// STORYFORGE_CONTENT environment variable, then a content/ directory relative
// to the current working directory.
func Load(override string) (*Config, error) {
	root := override
	if root == "" {
		root = os.Getenv("STORYFORGE_CONTENT")
	}
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting working directory: %w", err)
		}
		candidate := filepath.Join(cwd, defaultContentDir)
		if _, err := os.Stat(candidate); err == nil {
			root = candidate
		}
	}
	if root == "" {
		return nil, fmt.Errorf(
			"content directory not found; set STORYFORGE_CONTENT or run from a storyforge project directory",
		)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolving content path: %w", err)
	}
	return &Config{ContentRoot: abs}, nil
}