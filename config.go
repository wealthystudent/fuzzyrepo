package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config defines all configuration options.
//
// Put defaults in DefaultConfig(), and enforce rules in Validate().
// That way adding new fields stays simple.
type Config struct {
	LocalRepoPath string `yaml:"local_repo_path"`
	Affiliation   string `yaml:"affiliation"`
	MaxResults    int    `yaml:"max_results"`
}

// Returns the default config
func DefaultConfig() Config {
	return Config{
		LocalRepoPath: "",
		Affiliation:   "owner",
		MaxResults:    100,
	}
}

// LoadConfig reads YAML from disk, merges it onto defaults, and validates.
func LoadConfig() (Config, error) {
	path := DefaultConfigPath()

	cfg := DefaultConfig()

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// config file is optional
			return cfg, nil
		}
		return Config{}, err
	}

	// Unmarshal onto cfg so defaults remain for fields not in the file.
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}

	return cfg, nil
}

func (c Config) Validate() error {
	// Config.Affiliation
	if c.Affiliation == "" {
		return errors.New("affiliation cannot be empty")
	}
	switch c.Affiliation {
	case "owner", "collaborator", "organization":
		// ok
	default:
		return fmt.Errorf("affiliation must be one of: owner, collaborator, organization (got %q)", c.Affiliation)
	}

	// Config.MaxResults
	if c.MaxResults < 0 {
		return fmt.Errorf("max_results must be >= 0 (got %d)", c.MaxResults)
	}
	return nil
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".fuzzyrepo.yaml"
	}
	return filepath.Join(home, ".fuzzyrepo.yaml")
}
