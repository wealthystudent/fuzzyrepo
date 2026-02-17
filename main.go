package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Handle --sync-remote flag for background sync mode
	if len(os.Args) > 1 && os.Args[1] == "--sync-remote" {
		runRemoteSync()
		return
	}

	// Check if this is first run (no config file exists)
	firstRun := IsFirstRun()

	// On first run, check dependencies before proceeding
	if firstRun {
		if err := CheckDependencies(); err != nil {
			fmt.Fprintln(os.Stderr, "Setup error:", err)
			os.Exit(1)
		}
	}

	config, err := LoadConfig()
	if err != nil {
		log.Fatal("could not load config: ", err)
	}

	uiMsgs := make(chan tea.Msg, 10)
	refreshChan := make(chan struct{}, 1)

	initial, err := loadRepoCache()
	if err != nil {
		log.Println("Warning: could not load repo cache:", err)
	}

	// Load metadata to check sync status
	metadata, _ := LoadMetadata()
	cacheEmpty := len(initial) == 0
	needsRemoteSync := cacheEmpty || IsRemoteSyncDue(metadata)
	needsLocalScan := IsLocalScanDue(metadata)

	// If local scan is due, run it inline (fast) before showing UI
	// This ensures local repos are always up-to-date
	if needsLocalScan && len(config.GetRepoRoots()) > 0 {
		if updated, err := runLocalScan(config, initial); err == nil {
			initial = updated
		}
	}

	// Store initial cache mtime for change detection
	initialMtime := GetCacheMtime()

	// If remote sync is due (and not first run - we'll spawn after config is set)
	// spawn detached background process
	syncSpawned := false
	if needsRemoteSync && !firstRun && !isSyncRunning() {
		syncSpawned = spawnDetachedSync()
	}

	go func() {
		doRefresh := func() {
			uiMsgs <- refreshStartedMsg{}
			cfg, _ := LoadConfig()
			progressiveRefresh(cfg, uiMsgs)
		}

		// Manual refresh requests only - auto sync handled by detached process
		for range refreshChan {
			doRefresh()
		}
	}()

	selectedRepo, action, selectedPath, updatedConfig := ui(initial, config, uiMsgs, refreshChan, initialMtime, syncSpawned, firstRun)
	executeAction(selectedRepo, action, selectedPath, updatedConfig)
}

func executeAction(repo *Repository, action Action, selectedPath string, config Config) {
	if action == ActionNone {
		return
	}

	switch action {
	case ActionOpenPath:
		if strings.TrimSpace(selectedPath) == "" {
			return
		}
		if err := OpenInEditor(selectedPath, "manual"); err != nil {
			if errors.Is(err, ErrNoEditor) {
				fmt.Fprintln(os.Stderr, "Error: $EDITOR is not set")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Failed to open editor:", err)
			os.Exit(1)
		}
		return
	case ActionOpen:
		if repo == nil {
			return
		}
		localPath, err := EnsureLocal(*repo, config)
		if err != nil && !errors.Is(err, ErrAlreadyExists) {
			fmt.Fprintln(os.Stderr, "Clone failed:", err)
			os.Exit(1)
		}

		if err := OpenInEditor(localPath, repo.Name); err != nil {
			if errors.Is(err, ErrNoEditor) {
				fmt.Fprintln(os.Stderr, "Error: $EDITOR is not set")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Failed to open editor:", err)
			os.Exit(1)
		}

		_ = RecordUsage(*repo)

	case ActionCopy:
		if repo == nil {
			return
		}
		localPath, err := EnsureLocal(*repo, config)
		if err != nil && !errors.Is(err, ErrAlreadyExists) {
			fmt.Fprintln(os.Stderr, "Clone failed:", err)
			os.Exit(1)
		}

		CopyToClipboard(localPath)
		fmt.Println("Copied to clipboard:", localPath)

		_ = RecordUsage(*repo)

	case ActionBrowse:
		if repo == nil {
			return
		}
		if err := OpenInBrowser(*repo); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open browser:", err)
			os.Exit(1)
		}

		_ = RecordUsage(*repo)

	case ActionPRs:
		if repo == nil {
			return
		}
		if err := OpenPRs(*repo); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to open PRs:", err)
			os.Exit(1)
		}

		_ = RecordUsage(*repo)
	}
}
