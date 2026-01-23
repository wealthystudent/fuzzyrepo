package main

import (
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Cache for the repositories. Collection of RepoDTO pointers.
var repoCache []*RepoDTO

func main() {
	config, err := LoadConfig()
	if err != nil {
		log.Fatal("could not load config: ", err)
		os.Exit(1)
	}

	uiMsgs := make(chan tea.Msg, 10)

	var initial []RepoDTO

	if err := loadRepoJSONIntoCache(); err != nil {
		log.Println("Warning: could not load repo cache JSON:", err)
		repoCache = nil
	}
	initial = repoPtrsToValues(repoCache)

	go func() {
		uiMsgs <- refreshStartedMsg{}

		err := updateRepoJSON(config)
		if err != nil {
			log.Fatal("Could not update repos.json", err)
		}

		time.Sleep(8 * time.Second)

		var updated []RepoDTO
		if err := loadRepoJSONIntoCache(); err != nil {
			log.Println("Warning: could not load repo cache JSON:", err)
			repoCache = nil
		}
		updated = repoPtrsToValues(repoCache)

		uiMsgs <- reposUpdatedMsg(updated)

		uiMsgs <- refreshFinishedMsg{}
	}()

	ui(initial, uiMsgs)
}
