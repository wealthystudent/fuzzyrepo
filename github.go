package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func getAuthToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getGithubClient(ctx context.Context) (*github.Client, error) {
	token, err := getAuthToken()
	if err != nil {
		return nil, fmt.Errorf("not logged into gh: %w", err)
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := oauth2.NewClient(ctx, ts)

	return github.NewClient(client), nil
}

func getRemoteRepositories(ctx context.Context, githubClient *github.Client, cfg Config) ([]Repository, error) {
	opts := &github.RepositoryListOptions{
		Visibility:  "all",
		Affiliation: cfg.GitHub.Affiliation,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	var repos []Repository

	for {
		remoteRepos, resp, err := githubClient.Repositories.List(ctx, "", opts)
		if err != nil {
			return nil, err
		}

		for _, repo := range remoteRepos {
			owner := repo.GetOwner().GetLogin()
			name := repo.GetName()

			r := Repository{
				Owner:       owner,
				Name:        name,
				FullName:    owner + "/" + name,
				SSHURL:      repo.GetSSHURL(),
				LocalPath:   "",
				ExistsLocal: false,
			}
			r.ComputeSearchText()
			repos = append(repos, r)
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage

		if cfg.MaxResults > 0 && len(repos) >= cfg.MaxResults {
			break
		}
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
