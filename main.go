package main

import (
	"errors"
	"os"
)

// Used as a placeholder for when error handeling is not implemented yet.
var ErrNotImplemented = errors.New("not implemented yet")

// Used to store a list of the repositories ()
var repos *[]string

// Defines a DTO for a single repo (used to hold information about each repo)
type RepoDTO struct {
	name         string
	exists_local bool
}

// TODO: Function for retrieving a list of all repositories at the remote location
func listRemoteRepositories() error {
	searchResults := make([]string, 0)
	// TODO: Add code for retrieving the repos and append the names to searchResults

	repos = &searchResults
	return ErrNotImplemented
}

func main() {
	// Set repos using mock variable untill listRemoteRepositories is implemented
	repos = &mock_repos

	// Parse CLI (NOTE: RunCLI returns "int" os values)
	os.Exit(RunCLI(os.Args[1:]))
}
