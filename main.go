package main

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Toggle until your JSON updater/loader is implemented.
const (
	USE_MOCKS = false
)

// Cache for the repositories. Collection of RepoDTO pointers.
// (Keeping your shape so you can later populate it from JSON.)
var repoCache []*RepoDTO

func main() {
	uiMsgs := make(chan tea.Msg, 10)

	// ---- Initial list (fast) ----
	var initial []RepoDTO

	if USE_MOCKS {
		initial = mockRepos()
	} else {
		// Future: load from JSON into repoCache
		if err := loadRepoJSONIntoCache(); err != nil {
			log.Println("Warning: could not load repo cache JSON:", err)
			repoCache = nil
		}
		initial = repoPtrsToValues(repoCache)
	}

	// ---- Background refresh simulation ----
	go func() {
		uiMsgs <- refreshStartedMsg{}

		// Future: this is where you'd run your slow updater that writes JSON:
		// _ = updateRepoJSON(conf.LocalRepoPath)
		time.Sleep(8 * time.Second)

		var updated []RepoDTO
		if USE_MOCKS {
			updated = append(mockRepos(), mockReposMore()...)
		} else {
			if err := loadRepoJSONIntoCache(); err != nil {
				log.Println("Warning: could not load repo cache JSON:", err)
				repoCache = nil
			}
			updated = repoPtrsToValues(repoCache)
		}

		uiMsgs <- reposUpdatedMsg(updated)

		uiMsgs <- refreshFinishedMsg{}
	}()

	// ---- Start UI ----
	ui(initial, uiMsgs)
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

// ---- Mock data helpers ----

func mockRepos() []RepoDTO {
	return []RepoDTO{
		{Name: "my-local-repo", ExistsLocal: true, Path: "/Users/REDACTED/code/my-local-repo"},
		{Name: "my-remote-repo", ExistsLocal: false, Path: "git@github.com:wealthystudent/my-remote-repo.git"},
		{Name: "org-service-api", ExistsLocal: false, Path: "git@github.com:myorg/service-api.git"},
		{Name: "org-service-api (local)", ExistsLocal: true, Path: "/Users/REDACTED/code/service-api"},
	}
}

func mockReposMore() []RepoDTO {
	return []RepoDTO{
		{Name: "testingrepo", ExistsLocal: false, Path: "git@github.com:wealthystudent/testingrepo.git"},
		{Name: "testingrepo2", ExistsLocal: false, Path: "git@github.com:wealthystudent/testingrepo2.git"},
	}
}
