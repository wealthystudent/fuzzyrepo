package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/go-github/v68/github"
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
	var allRepos []Repository

	// Parse affiliations from config
	affiliations := parseAffiliations(cfg.GitHub.Affiliation)

	// Fetch repos for each affiliation separately to track the affiliation type
	for _, affiliation := range affiliations {
		repos, err := fetchReposWithAffiliation(ctx, githubClient, affiliation)
		if err != nil {
			return nil, err
		}
		allRepos = append(allRepos, repos...)
	}

	// Deduplicate repos (a repo might appear in multiple affiliations)
	return deduplicateRepos(allRepos), nil
}

// parseAffiliations splits the affiliation string into individual types
func parseAffiliations(affiliation string) []string {
	var result []string
	for _, a := range strings.Split(affiliation, ",") {
		a = strings.TrimSpace(a)
		if a != "" {
			result = append(result, a)
		}
	}
	return result
}

// fetchReposWithAffiliation fetches repos for a single affiliation type
func fetchReposWithAffiliation(ctx context.Context, githubClient *github.Client, affiliation string) ([]Repository, error) {
	opts := &github.RepositoryListOptions{
		Visibility:  "all",
		Affiliation: affiliation,
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
				Affiliation: affiliation,
			}
			r.ComputeSearchText()
			repos = append(repos, r)
		}

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return repos, nil
}

// deduplicateRepos removes duplicate repos, keeping the first occurrence
// (which preserves the affiliation priority: owner > collaborator > org_member)
func deduplicateRepos(repos []Repository) []Repository {
	seen := make(map[string]bool)
	var result []Repository

	for _, repo := range repos {
		if !seen[repo.FullName] {
			seen[repo.FullName] = true
			result = append(result, repo)
		}
	}

	return result
}
