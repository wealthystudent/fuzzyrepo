package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatal("could not load config: ", err)
		os.Exit(1)
	}

	uiMsgs := make(chan tea.Msg, 10)
	refreshChan := make(chan struct{}, 1)

	initial, err := loadRepoCache()
	if err != nil {
		log.Println("Warning: could not load repo cache:", err)
	}

	go func() {
		doRefresh := func() {
			uiMsgs <- refreshStartedMsg{}

			cfg, _ := LoadConfig()
			err := updateRepoCache(cfg)
			if err != nil {
				log.Println("Warning: could not update repo cache:", err)
			}

			updated, err := loadRepoCache()
			if err != nil {
				log.Println("Warning: could not load repo cache:", err)
			}

			uiMsgs <- reposUpdatedMsg(updated)
			uiMsgs <- refreshFinishedMsg{}
		}

		doRefresh()

		for range refreshChan {
			doRefresh()
		}
	}()

	selectedRepo, action := ui(initial, config, uiMsgs, refreshChan)
	executeAction(selectedRepo, action, config)
}

func executeAction(repo *Repository, action Action, config Config) {
	if repo == nil || action == ActionNone {
		return
	}

	switch action {
	case ActionOpen:
		localPath, err := EnsureLocal(*repo, config)
		if err != nil && !errors.Is(err, ErrAlreadyExists) {
			fmt.Fprintln(os.Stderr, "Clone failed:", err)
			os.Exit(1)
		}

		if err := OpenInEditor(localPath); err != nil {
			if errors.Is(err, ErrNoEditor) {
				fmt.Fprintln(os.Stderr, "Error: $EDITOR is not set")
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "Failed to open editor:", err)
			os.Exit(1)
		}

	case ActionCopy:
		localPath, err := EnsureLocal(*repo, config)
		if err != nil && !errors.Is(err, ErrAlreadyExists) {
			fmt.Fprintln(os.Stderr, "Clone failed:", err)
			os.Exit(1)
		}

		CopyToClipboard(localPath)
		fmt.Println("Copied to clipboard:", localPath)
	}
}
