package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	LocalRepoPath string
}

func LoadConfig() (*Config, error) {
	configPath := DefaultConfigPath()
	b, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, err
	}

	line := strings.TrimSpace(string(b))
	const prefix = `local-repo-directory:`
	if !strings.HasPrefix(line, prefix) {
		return nil, fmt.Errorf("invalid config: expected %q", prefix)
	}

	val := strings.TrimSpace(strings.TrimPrefix(line, prefix))
	val = strings.Trim(val, `"'`) // remove optional quotes

	return &Config{LocalRepoPath: val}, nil
}

func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fuzzyrepo.conf"), nil
}

func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".fuzzyrepo.json" // fallback
	}
	return filepath.Join(home, ".fuzzyrepo.json")
}
