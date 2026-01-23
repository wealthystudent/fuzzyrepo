package main

import (
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

	initial, err := loadRepoCache()
	if err != nil {
		log.Println("Warning: could not load repo cache:", err)
	}

	go func() {
		uiMsgs <- refreshStartedMsg{}

		err := updateRepoCache(config)
		if err != nil {
			log.Println("Warning: could not update repo cache:", err)
		}

		updated, err := loadRepoCache()
		if err != nil {
			log.Println("Warning: could not load repo cache:", err)
		}

		uiMsgs <- reposUpdatedMsg(updated)
		uiMsgs <- refreshFinishedMsg{}
	}()

	ui(initial, uiMsgs)
}
