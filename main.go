package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
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

func getAuthToken() (string, error) {
	// Uses the gh CLI to get the authentication token
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getOauth2Client(ctx context.Context, token string) (*http.Client, error) {
	// Creates an OAuth2 client using the provided token
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return oauth2.NewClient(ctx, ts), nil
}

func getGithubClient(client *http.Client) *github.Client {
	// Creates a GitHub client using the provided HTTP client
	return github.NewClient(client)
}

func getRemoteRepositories(ctx context.Context, githubClient *github.Client) error {
	// Fetches remote repositories and populates the repoCache

	opts := &github.RepositoryListOptions{
		Visibility:  "all",                                    // Can be "public", "private", or "all"
		Affiliation: "owner,collaborator,organization_member", // Include repos from all affiliations
		ListOptions: github.ListOptions{PerPage: 100},         // Fetch maximum items per page
	}
	remote_repos, _, err := githubClient.Repositories.List(ctx, "", opts)
	if err != nil {
		return err
	}
	for _, repo := range remote_repos {
		dto := &RepoDTO{
			name:         repo.GetName(),
			url:          repo.GetSSHURL(),
			exists_local: false,
		} // Initialize RepoDTO, as value
		repoCache = append(repoCache, dto) // Append the DTO pointer to the repoCache
	}
	return nil
}

func main() {
	ctx := context.Background()
	token, err := getAuthToken()

	if err != nil {
		log.Fatal("Not logged into gh: ", err)
		panic(err)
	}

	client, err := getOauth2Client(ctx, token)

	if err != nil {
		log.Fatal("Failed to create OAuth2 client: ", err)
		panic(err)
	}

	githubClient := getGithubClient(client)

	err = getRemoteRepositories(ctx, githubClient)
	if err != nil {
		log.Fatal("Failed to get remote repositories: ", err)
		panic(err)
	}

	for i, r := range repoCache {
		// r is a *RepoDTO (a pointer)
		// Go automatically handles the pointer so you can just use the dot (.)
		fmt.Printf("%d. Name: %s | URL: %s | Local: %v\n", i+1, r.name, r.url, r.exists_local)
	}

}
