package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

type GitHubConfig struct {
	Affiliation string `yaml:"affiliation"`
	Orgs        string `yaml:"orgs"`
}

type Config struct {
	RepoRoots  []string     `yaml:"repo_roots"`
	CloneRoot  string       `yaml:"clone_root"`
	GitHub     GitHubConfig `yaml:"github"`
	MaxResults int          `yaml:"max_results"`
}

func DefaultConfig() Config {
	return Config{
		RepoRoots: nil,
		CloneRoot: "",
		GitHub: GitHubConfig{
			Affiliation: "owner,collaborator,organization_member",
			Orgs:        "",
		},
		MaxResults: 0,
	}
}

func (c Config) GetRepoRoots() []string {
	return c.RepoRoots
}

func (c Config) GetCloneRoot() string {
	if c.CloneRoot != "" {
		return c.CloneRoot
	}
	roots := c.GetRepoRoots()
	if len(roots) > 0 {
		return roots[0]
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "repos")
}

func (c Config) GetOrgs() []string {
	if c.GitHub.Orgs == "" {
		return nil
	}
	parts := strings.Split(c.GitHub.Orgs, ",")
	var orgs []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			orgs = append(orgs, p)
		}
	}
	return orgs
}

func LoadConfig() (Config, error) {
	cfg := DefaultConfig()

	xdgPath := xdgConfigPath()
	legacyPath := legacyConfigPath()

	var configPath string
	if _, err := os.Stat(xdgPath); err == nil {
		configPath = xdgPath
	} else if _, err := os.Stat(legacyPath); err == nil {
		configPath = legacyPath
	} else {
		return cfg, nil
	}

	b, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", configPath, err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", configPath, err)
	}

	return cfg, nil
}

func SaveConfig(cfg Config) error {
	configPath := xdgConfigPath()
	configDir := filepath.Dir(configPath)

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}

	b, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	tmpPath := configPath + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, configPath)
}

func (c Config) Validate() error {
	if c.GitHub.Affiliation == "" {
		return errors.New("github.affiliation cannot be empty")
	}

	if c.MaxResults < 0 {
		return fmt.Errorf("max_results must be >= 0 (got %d)", c.MaxResults)
	}

	for _, root := range c.RepoRoots {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("repo_roots must contain absolute paths (got %q)", root)
		}
	}

	if c.CloneRoot != "" && !filepath.IsAbs(c.CloneRoot) {
		return fmt.Errorf("clone_root must be an absolute path (got %q)", c.CloneRoot)
	}

	return nil
}

func xdgConfigPath() string {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, "AppData", "Roaming")
		}
	default:
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, ".config")
		}
	}

	return filepath.Join(configDir, "fuzzyrepo", "config.yaml")
}

func legacyConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".fuzzyrepo.yaml"
	}
	return filepath.Join(home, ".fuzzyrepo.yaml")
}

func ConfigPath() string {
	xdgPath := xdgConfigPath()
	legacyPath := legacyConfigPath()

	if _, err := os.Stat(xdgPath); err == nil {
		return xdgPath
	}
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath
	}
	return xdgPath
}
