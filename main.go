package main

import (
	"context"
	"fmt"
	"log"
	"os"
)

// Cache for the repositories. Collection of RepoDTO pointers.
var repoCache []*RepoDTO
var localRepoCache []*RepoDTO

func main() {
	ctx := context.Background()

	// Create GH Client
	githubClient, err := getGithubClient(ctx)
	if err != nil {
		log.Fatal("Failed to create github client: ", err)
		panic(err)
	}

	// Read config
	conf, err := LoadConfig()
	if err != nil {
		log.Fatal("Failed to read config file. Make sure to setup the ~/.fuzzyrepo.conf file: ", err)
		panic(err)
	}

	// Get repositories
	err = getRemoteRepositories(ctx, githubClient)
	if err != nil {
		log.Fatal("Failed to get remote repositories: ", err)
		panic(err)
	}

	// Get local repositories (look for .git fuyzzyfind)
	err_local_repo := getClonedRepos(conf.LocalRepoPath)
	if err_local_repo != nil {
		log.Fatal("Failed to retrive local repositories: ", err)
		panic(err)
	}

	fmt.Println("Local Repos Found")
	for i, r := range localRepoCache {
		// r is a *RepoDTO (a pointer)
		// Go automatically handles the pointer so you can just use the dot (.)
		fmt.Printf("%d. folder_path: %s \n", i+1, r.url)
	}

	// Parse CLI: Entry point for the CLI tool
	// (NOTE: RunCLI returns "int" os values)
	os.Exit(RunCLI(os.Args[1:]))

}
