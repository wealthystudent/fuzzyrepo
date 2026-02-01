package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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

func progressiveRefresh(config Config, uiMsgs chan<- tea.Msg) {
	ctx := context.Background()
	cacheDir := getCacheDir()
	path := getCachePath()

	_ = os.MkdirAll(cacheDir, 0o755)

	localRepos := indexLocalRepos(config.GetRepoRoots())

	githubClient, err := getGithubClient(ctx)
	if err != nil {
		uiMsgs <- errorMsg{err: err}
		uiMsgs <- refreshFinishedMsg{}
		return
	}

	remoteRepos, err := getRemoteRepositories(ctx, githubClient, config)
	if err != nil {
		uiMsgs <- errorMsg{err: err}
		uiMsgs <- refreshFinishedMsg{}
		return
	}

	merged := mergeRepos(localRepos, remoteRepos)

	uiMsgs <- reposUpdatedMsg(merged)

	b, err := json.MarshalIndent(merged, "", "  ")
	if err == nil {
		tmpPath := path + ".tmp"
		if err := os.WriteFile(tmpPath, b, 0o600); err == nil {
			_ = os.Rename(tmpPath, path)
		}
	}

	// Update metadata with sync timestamps
	meta, _ := LoadMetadata()
	meta.UpdateRemoteSyncTime()
	meta.UpdateLocalScanTime()
	_ = SaveMetadata(meta)

	uiMsgs <- refreshFinishedMsg{}
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

func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

// padLineToWidth pads a line to the specified width using the given style
// This ensures the background color extends to the full terminal width
func padLineToWidth(line string, width int, style lipgloss.Style) string {
	lineWidth := lipgloss.Width(line)
	if lineWidth >= width {
		return line
	}
	return line + style.Render(strings.Repeat(" ", width-lineWidth))
}
