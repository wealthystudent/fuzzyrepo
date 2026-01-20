package main

import (
	"context"
	"errors"
	"log"
	"os/exec"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
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

func getOauth2Client(token string) (*oauth2.Client, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return oauth2.NewClient(context.Background(), ts), nil
}

func getAuthToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func getGithubClient(client *oauth2.Client) *github.Client {
	return github.NewClient(client)
}

// TODO: Function for retrieving a list of all repositories at the remote location
func listRemoteRepositories() error {
	searchResults := make([]string, 0)
	// TODO: Add code for retrieving the repos and append the names to searchResults

	repos = &searchResults
	return ErrNotImplemented
}

func listRemoteOrganizations() error {
	return ErrNotImplemented
}

func listPublicRepositories() error {
	return ErrNotImplemented
}

func main() {
	token, err := getAuthToken()

	if err != nil {
		log.Fatal("Not logged into gh: ", err)
		panic(err)
	}

	client, err = getOauth2Client(token)

	if err != nil {
		log.Fatal("Failed to create OAuth2 client: ", err)
		panic(err)
	}

	githubClient := getGithubClient(client)

}
