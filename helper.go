package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func getOs() string {
	// get runtime OS
	return runtime.GOOS
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// Makes a string exactly w characters long by cutting it down (adding “…” if needed) or padding it with spaces.
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

// Constrains v to the inclusive range [lo, hi] (returns lo if below, hi if above).
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func updateRepoJSON(config Config) error {
	ctx := context.Background()
	cacheDir := filepath.Join(getHomeDir(), ".local", "share", "fuzzyrepo")
	path := filepath.Join(cacheDir, "repos.json")

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	githubClient, err := getGithubClient(ctx)
	if err != nil {
		log.Fatal("Failed to create github client: ", err)
		return err
	}

	repos, err := getRemoteRepositories(ctx, githubClient)
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

func loadRepoJSONIntoCache() error {
	cacheDir := filepath.Join(getHomeDir(), ".local", "share", "fuzzyrepo")
	path := filepath.Join(cacheDir, "repos.json")

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var repos []*RepoDTO
	if err := json.Unmarshal(data, &repos); err != nil {
		return err
	}

	repoCache = repos
	return nil
}

func repoPtrsToValues(in []*RepoDTO) []RepoDTO {
	out := make([]RepoDTO, 0, len(in))
	for _, p := range in {
		if p == nil {
			continue
		}
		out = append(out, *p) // copy value into UI slice (avoids sharing/mutation issues)
	}
	return out
}
