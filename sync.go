package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
)

// Sync lock file path
func getSyncLockPath() string {
	return filepath.Join(getCacheDir(), "sync.lock")
}

// isSyncRunning checks if a sync process is currently running
// Returns true if lock file exists and the PID is still alive
func isSyncRunning() bool {
	lockPath := getSyncLockPath()
	data, err := os.ReadFile(lockPath)
	if err != nil {
		return false // No lock file means no sync running
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		// Invalid lock file, remove it
		_ = os.Remove(lockPath)
		return false
	}

	// Check if process is still running
	if !isProcessRunning(pid) {
		// Stale lock file, remove it
		_ = os.Remove(lockPath)
		return false
	}

	return true
}

// isProcessRunning checks if a process with the given PID is running
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	// to check if the process actually exists
	if runtime.GOOS != "windows" {
		err = process.Signal(syscall.Signal(0))
		return err == nil
	}

	// On Windows, FindProcess only succeeds if the process exists
	return true
}

// acquireSyncLock creates a lock file with the current PID
// Returns true if lock was acquired, false if sync is already running
func acquireSyncLock() bool {
	if isSyncRunning() {
		return false
	}

	lockPath := getSyncLockPath()
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false
	}

	pid := os.Getpid()
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		return false
	}

	return true
}

// releaseSyncLock removes the lock file
func releaseSyncLock() {
	_ = os.Remove(getSyncLockPath())
}

// runRemoteSync performs a full remote sync operation
// This is called when fuzzyrepo is invoked with --sync-remote flag
func runRemoteSync() {
	// Try to acquire lock
	if !acquireSyncLock() {
		fmt.Fprintln(os.Stderr, "Another sync is already running")
		os.Exit(1)
	}
	defer releaseSyncLock()

	// Load config
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to load config:", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Get GitHub client
	githubClient, err := getGithubClient(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to get GitHub client:", err)
		os.Exit(1)
	}

	// Fetch remote repositories
	remoteRepos, err := getRemoteRepositories(ctx, githubClient, config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to fetch remote repos:", err)
		os.Exit(1)
	}

	// Scan local repositories
	localRepos := indexLocalRepos(config.GetRepoRoots())

	// Merge repos
	merged := mergeRepos(localRepos, remoteRepos)

	// Save to cache
	cacheDir := getCacheDir()
	cachePath := getCachePath()

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to create cache dir:", err)
		os.Exit(1)
	}

	b, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to marshal repos:", err)
		os.Exit(1)
	}

	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write cache:", err)
		os.Exit(1)
	}

	if err := os.Rename(tmpPath, cachePath); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to rename cache:", err)
		os.Exit(1)
	}

	// Update metadata
	meta, _ := LoadMetadata()
	meta.UpdateRemoteSyncTime()
	meta.UpdateLocalScanTime()
	if err := SaveMetadata(meta); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save metadata:", err)
		// Don't exit, cache was saved successfully
	}

	fmt.Printf("Synced %d repositories\n", len(merged))
}

// spawnDetachedSync starts a background sync process that continues even after
// the main process exits. Returns true if spawn was successful.
func spawnDetachedSync() bool {
	// Check if sync is already running
	if isSyncRunning() {
		return false
	}

	// Get the path to our own executable
	executable, err := os.Executable()
	if err != nil {
		return false
	}

	// Create the command
	cmd := exec.Command(executable, "--sync-remote")

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}

	// Redirect output to null (we don't want to see it)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	// Start the process (don't wait for it)
	if err := cmd.Start(); err != nil {
		return false
	}

	// Detach - we don't call cmd.Wait() so the process continues independently
	return true
}

// runLocalScan performs a local-only scan (no GitHub API calls)
// This is fast and can be run inline on startup
// Returns the merged repos and any error
func runLocalScan(config Config, existingRepos []Repository) ([]Repository, error) {
	// Scan local repositories
	localRepos := indexLocalRepos(config.GetRepoRoots())

	// Merge with existing cached repos
	merged := mergeRepos(localRepos, existingRepos)

	// Save to cache
	if err := saveReposToCache(merged); err != nil {
		return merged, err
	}

	// Update metadata (only local scan time)
	meta, _ := LoadMetadata()
	meta.UpdateLocalScanTime()
	_ = SaveMetadata(meta)

	return merged, nil
}

// saveReposToCache saves the repos slice to the cache file atomically
func saveReposToCache(repos []Repository) error {
	cacheDir := getCacheDir()
	cachePath := getCachePath()

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}

	b, err := json.MarshalIndent(repos, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := cachePath + ".tmp"
	if err := os.WriteFile(tmpPath, b, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, cachePath)
}
