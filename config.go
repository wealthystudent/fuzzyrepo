package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"gopkg.in/yaml.v3"
)

type GitHubConfig struct {
	Affiliation string `yaml:"affiliation"`
	Orgs        string `yaml:"orgs"`
}

// CloneRule defines a regex pattern to match repo full_name and a target directory
type CloneRule struct {
	Pattern string `yaml:"pattern"` // Regex pattern to match against full_name (owner/repo)
	Path    string `yaml:"path"`    // Target directory (repo name will be appended)
}

// ConfigFieldDescriptions maps config field indices to their descriptions
// Used in the config overlay to show help text for the focused field
var ConfigFieldDescriptions = map[int]string{
	0: "Directories to scan for local git repositories (comma-separated absolute paths)",
	1: "Default clone directory when clone rules are disabled or no rule matches",
	2: "Enable regex-based clone rules. Press 'e' to edit config file and add rules:\n  clone_rules:\n    - pattern: \"^org/.*\"\n      path: /path/to/dir",
	3: "Limit to specific GitHub organizations (comma-separated, empty = all orgs)",
	4: "Show repositories you own (yes/no)",
	5: "Show repositories you collaborate on (yes/no)",
	6: "Show repositories from your organizations (yes/no)",
	7: "Show local-only repositories not on GitHub (yes/no)",
}

type Config struct {
	RepoRoots     []string     `yaml:"repo_roots"`
	CloneRoot     string       `yaml:"clone_root"`
	UseCloneRules bool         `yaml:"use_clone_rules"`       // Enable regex-based clone path rules
	CloneRules    []CloneRule  `yaml:"clone_rules,omitempty"` // Ordered rules for clone path, first match wins
	GitHub        GitHubConfig `yaml:"github"`

	// Filter settings - control which repos are displayed from cache
	ShowOwner        bool `yaml:"show_owner"`        // Show repos owned by user (default true)
	ShowCollaborator bool `yaml:"show_collaborator"` // Show repos user is collaborator on (default true)
	ShowOrgMember    bool `yaml:"show_org_member"`   // Show repos from orgs user is member of (default true)
	ShowLocal        bool `yaml:"show_local"`        // Show local-only repos (default true)
}

func DefaultConfig() Config {
	return Config{
		RepoRoots:     nil,
		CloneRoot:     "",
		UseCloneRules: false,
		CloneRules:    nil,
		GitHub: GitHubConfig{
			Affiliation: "owner,collaborator,organization_member",
			Orgs:        "",
		},
		ShowOwner:        true,
		ShowCollaborator: true,
		ShowOrgMember:    true,
		ShowLocal:        true,
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

// GetClonePath returns the full destination path for cloning a repo.
// If UseCloneRules is enabled, it checks clone_rules in order and returns the first matching rule's path + repo name.
// Falls back to clone_root + repo name if no rules match or UseCloneRules is disabled.
func (c Config) GetClonePath(fullName, repoName string) string {
	if c.UseCloneRules {
		for _, rule := range c.CloneRules {
			re, err := regexp.Compile(rule.Pattern)
			if err != nil {
				// Invalid regex, skip this rule
				continue
			}
			if re.MatchString(fullName) {
				return filepath.Join(rule.Path, repoName)
			}
		}
	}
	// No rules matched or UseCloneRules disabled, use default clone root
	return filepath.Join(c.GetCloneRoot(), repoName)
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

	for _, root := range c.RepoRoots {
		if !filepath.IsAbs(root) {
			return fmt.Errorf("repo_roots must contain absolute paths (got %q)", root)
		}
	}

	if c.CloneRoot != "" && !filepath.IsAbs(c.CloneRoot) {
		return fmt.Errorf("clone_root must be an absolute path (got %q)", c.CloneRoot)
	}

	// Validate clone rules
	for i, rule := range c.CloneRules {
		if rule.Pattern == "" {
			return fmt.Errorf("clone_rules[%d]: pattern cannot be empty", i)
		}
		if _, err := regexp.Compile(rule.Pattern); err != nil {
			return fmt.Errorf("clone_rules[%d]: invalid regex pattern %q: %v", i, rule.Pattern, err)
		}
		if rule.Path == "" {
			return fmt.Errorf("clone_rules[%d]: path cannot be empty", i)
		}
		if !filepath.IsAbs(rule.Path) {
			return fmt.Errorf("clone_rules[%d]: path must be absolute (got %q)", i, rule.Path)
		}
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

// IsFirstRun returns true if this is the first time fuzzyrepo is being run
// (no config file exists)
func IsFirstRun() bool {
	xdgPath := xdgConfigPath()
	legacyPath := legacyConfigPath()

	if _, err := os.Stat(xdgPath); err == nil {
		return false
	}
	if _, err := os.Stat(legacyPath); err == nil {
		return false
	}
	return true
}

// CheckDependencies verifies that all required dependencies are installed and configured
// Returns an error with a helpful message if any dependency is missing
func CheckDependencies() error {
	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("git is not installed. Please install git and try again")
	}

	// Check if gh (GitHub CLI) is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New("GitHub CLI (gh) is not installed. Please install it: https://cli.github.com")
	}

	// Check if gh is authenticated
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return errors.New("GitHub CLI is not authenticated. Please run: gh auth login")
	}

	return nil
}
