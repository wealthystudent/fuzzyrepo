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

	remoteRepos, err := getRemoteRepositories(ctx, githubClient, config)
	if err != nil {
		return err
	}

	localRepos := indexLocalRepos(config.GetRepoRoots())

	merged := mergeRepos(localRepos, remoteRepos)

	b, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func mergeRepos(local, remote []Repository) []Repository {
	repoMap := make(map[string]Repository)

	for _, r := range remote {
		key := strings.ToLower(r.FullName)
		repoMap[key] = r
	}

	for _, r := range local {
		key := strings.ToLower(r.FullName)
		if existing, ok := repoMap[key]; ok {
			existing.LocalPath = r.LocalPath
			existing.ExistsLocal = true
			existing.ComputeSearchText()
			repoMap[key] = existing
		} else {
			repoMap[key] = r
		}
	}

	result := make([]Repository, 0, len(repoMap))
	for _, r := range repoMap {
		result = append(result, r)
	}

	return result
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
