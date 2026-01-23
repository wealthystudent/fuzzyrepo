package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func getOs() string {
	return runtime.GOOS
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

func padOrTrim(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) > w {
		if w <= 1 {
			return s[:w]
		}
		return s[:w-1] + "…"
	}
	return s + strings.Repeat(" ", w-len(s))
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func getCacheDir() string {
	return filepath.Join(getHomeDir(), ".local", "share", "fuzzyrepo")
}

func getCachePath() string {
	return filepath.Join(getCacheDir(), "repos.json")
}

func updateRepoCache(config Config) error {
	ctx := context.Background()
	cacheDir := getCacheDir()
	path := getCachePath()

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	githubClient, err := getGithubClient(ctx)
	if err != nil {
		return err
	}

	repos, err := getRemoteRepositories(ctx, githubClient, config)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func loadRepoCache() ([]Repository, error) {
	cacheDir := getCacheDir()
	path := getCachePath()

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var repos []Repository
	if err := json.Unmarshal(data, &repos); err != nil {
		return nil, err
	}

	for i := range repos {
		repos[i].ComputeSearchText()
	}

	return repos, nil
}
