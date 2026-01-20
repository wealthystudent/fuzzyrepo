package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Helper function to handle the file scanning
func extractURLFromConfig(configPath string) (string, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "url =") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return "", fmt.Errorf("url not found")
}

func createLocalRepoCache(localRepos []string) error {
	// Append git remote url to localRepoCache
	for _, folderPath := range localRepos {

		configPath := filepath.Join(folderPath, "config")

		url, err := extractURLFromConfig(configPath)
		if err != nil {
			continue
		}

		dto := &LocalRepoDTO{
			url:        url,
			folderPath: folderPath,
		}
		localRepoCache = append(localRepoCache, dto) // Append the DTO pointer to the repoCache

		// Initialize RepoDTO, as value

	}

	return nil
}

func getClonedRepos(searchPath string) error {
	// find git folders on local computer
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows-specific behavior
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Get-ChildItem -Path '%s' -Filter '.git' -Recurse -Hidden -Directory -ErrorAction SilentlyContinue | Select-Object -ExpandProperty FullName", searchPath))
	case "darwin", "linux":
		// macOS behavior
		cmd = exec.Command("find", searchPath, "-name", ".git", "-type", "d", "-prune")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	localRepos := strings.Split(string(output), "\n")

	if err != nil {
		return fmt.Errorf("Error finding repositories", err)
	}

	createLocalRepoCache(localRepos)

	// Process output to extract repository paths
	return nil
}
