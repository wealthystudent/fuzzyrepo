package main

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// Retrieves locally stored gh auth token
func getAuthToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Create github client
func getGithubClient(ctx context.Context) (*github.Client, error) {

	token, err := getAuthToken()
	if err != nil {
		log.Fatal("Not logged into gh: ", err)
		return nil, err
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := oauth2.NewClient(ctx, ts)

	return github.NewClient(client), nil
}

// Fetch remote repositories and return them as a list of RepoDTOs.
func getRemoteRepositories(ctx context.Context, githubClient *github.Client) ([]*RepoDTO, error) {

	opts := &github.RepositoryListOptions{
		Visibility:  "all",
		Affiliation: "owner,collaborator,organization_member",
		ListOptions: github.ListOptions{PerPage: 100},
	}

	repos := []*RepoDTO{}

	for {
		remoteRepos, resp, err := githubClient.Repositories.List(ctx, "", opts)
		if err != nil {
			return nil, err
		}
		// TODO: Fix exists local
		for _, repo := range remoteRepos {
			repos = append(repos, &RepoDTO{
				Name:        repo.GetName(),
				Path:        repo.GetSSHURL(),
				ExistsLocal: false,
			})
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return repos, nil
}

func cloneRepo(ctx context.Context, sshURL, localPath string) error {
	if strings.TrimSpace(sshURL) == "" {
		return fmt.Errorf("sshURL is empty")
	}
	if strings.TrimSpace(localPath) == "" {
		return fmt.Errorf("localPath is empty")
	}

	cmd := exec.CommandContext(ctx, "git", "clone", sshURL, localPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
	}
	return nil
}
