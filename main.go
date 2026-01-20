package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
)

// Used as a placeholder for when error handeling is not implemented yet.
var ErrNotImplemented = errors.New("not implemented yet")

// Defines a DTO for a single repo (used to hold information about each repo)
type RepoDTO struct {
	name         string
	url          string
	exists_local bool
}

// Cache for the repositories. Collection of RepoDTO pointers.
var repoCache []*RepoDTO

func main() {
	ctx := context.Background()

	// Create GH Client
	githubClient, err := getGithubClient(ctx)
	if err != nil {
		log.Fatal("Failed to create github client: ", err)
		panic(err)
	}

	// Get repositories
	err = getRemoteRepositories(ctx, githubClient)
	if err != nil {
		log.Fatal("Failed to get remote repositories: ", err)
		panic(err)
	}

	// Loop through repositories and print
	for i, r := range repoCache {
		// r is a *RepoDTO (a pointer)
		// Go automatically handles the pointer so you can just use the dot (.)
		fmt.Printf("%d. Name: %s | URL: %s | Local: %v\n", i+1, r.name, r.url, r.exists_local)
	}

	// Parse CLI: Entry point for the CLI tool
	// (NOTE: RunCLI returns "int" os values)
	os.Exit(RunCLI(os.Args[1:]))

}
