package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

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

func getGithubClient(ctx context.Context) (*github.Client, error) {
	// Creates a GitHub client using the provided HTTP client
	token, err := getAuthToken()

	if err != nil {
		log.Fatal("Not logged into gh: ", err)
		return nil, err
	}

	client, err := getOauth2Client(ctx, token)

	if err != nil {
		log.Fatal("Failed to create OAuth2 client: ", err)
		return nil, err
	}
	return github.NewClient(client), nil
}

func getRemoteRepositories(ctx context.Context, githubClient *github.Client) error {
	// Fetches remote repositories and populates the repoCache

	opts := &github.RepositoryListOptions{
		Visibility:  "all",                                    // Can be "public", "private", or "all"
		Affiliation: "owner,collaborator,organization_member", // Include repos from all affiliations
		ListOptions: github.ListOptions{PerPage: 100},         // Fetch maximum items per page
	}

	for {
		remote_repos, resp, err := githubClient.Repositories.List(ctx, "", opts)
		if err != nil {
			return err
		}
		for _, repo := range remote_repos {
			dto := &RepoDTO{
				Name:        repo.GetName(),
				Path:        repo.GetSSHURL(),
				ExistsLocal: false,
			} // Initialize RepoDTO, as value
			repoCache = append(repoCache, dto) // Append the DTO pointer to the repoCache
		}
		if resp.NextPage == 0 {
			break
		}
		fmt.Printf("Successfully loaded %d repos, page %d\n", len(repoCache), opts.Page)
		opts.Page = resp.NextPage
		if opts.Page == 4 {
			break
		}
	}

	return nil
}
