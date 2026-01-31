package main

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Repository struct {
	Owner       string `json:"owner"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	SSHURL      string `json:"ssh_url"`
	LocalPath   string `json:"local_path"`
	ExistsLocal bool   `json:"exists_local"`
	Affiliation string `json:"affiliation"` // "owner", "collaborator", "organization_member", "local"
	SearchText  string `json:"-"`
}

func (r *Repository) ComputeSearchText() {
	r.SearchText = strings.ToLower(r.Owner + " " + r.Name + " " + r.FullName)
}

func extractOriginURL(gitConfigPath string) (string, error) {
	file, err := os.Open(gitConfigPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	inOriginSection := false
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "[remote \"origin\"]") {
			inOriginSection = true
			continue
		}

		if inOriginSection {
			if strings.HasPrefix(line, "[") {
				break
			}
			if strings.HasPrefix(line, "url") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1]), nil
				}
			}
		}
	}

	return "", nil
}

var (
	sshURLPattern   = regexp.MustCompile(`git@github\.com:([^/]+)/(.+?)(?:\.git)?$`)
	httpsURLPattern = regexp.MustCompile(`https://github\.com/([^/]+)/(.+?)(?:\.git)?$`)
)

func parseGitHubURL(url string) (owner, name string, ok bool) {
	if matches := sshURLPattern.FindStringSubmatch(url); matches != nil {
		return matches[1], strings.TrimSuffix(matches[2], ".git"), true
	}
	if matches := httpsURLPattern.FindStringSubmatch(url); matches != nil {
		return matches[1], strings.TrimSuffix(matches[2], ".git"), true
	}
	return "", "", false
}

func indexLocalRepos(roots []string) []Repository {
	var repos []Repository

	for _, root := range roots {
		if root == "" {
			continue
		}

		filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if d.IsDir() && d.Name() == ".git" {
				repoPath := filepath.Dir(path)
				repo := buildRepoFromLocalPath(repoPath)
				repos = append(repos, repo)
				return fs.SkipDir
			}

			if d.IsDir() && (d.Name() == "node_modules" || d.Name() == "vendor" || d.Name() == ".cache") {
				return fs.SkipDir
			}

			return nil
		})
	}

	return repos
}

func buildRepoFromLocalPath(repoPath string) Repository {
	gitConfigPath := filepath.Join(repoPath, ".git", "config")
	originURL, _ := extractOriginURL(gitConfigPath)

	owner, name, ok := parseGitHubURL(originURL)
	if !ok {
		name = filepath.Base(repoPath)
		owner = "local"
	}

	repo := Repository{
		Owner:       owner,
		Name:        name,
		FullName:    owner + "/" + name,
		SSHURL:      originURL,
		LocalPath:   repoPath,
		ExistsLocal: true,
		Affiliation: "local",
	}
	repo.ComputeSearchText()

	return repo
}

// filterRepos filters the full repository cache based on config settings
// Returns a new slice containing only repos that match the filter criteria
func filterRepos(repos []Repository, cfg Config) []Repository {
	filtered := make([]Repository, 0, len(repos))

	for _, repo := range repos {
		if shouldIncludeRepo(repo, cfg) {
			filtered = append(filtered, repo)
		}
	}

	return filtered
}

// shouldIncludeRepo checks if a repo should be included based on config filters
func shouldIncludeRepo(repo Repository, cfg Config) bool {
	switch repo.Affiliation {
	case "owner":
		return cfg.ShowOwner
	case "collaborator":
		return cfg.ShowCollaborator
	case "organization_member":
		return cfg.ShowOrgMember
	case "local":
		return cfg.ShowLocal
	default:
		// Unknown affiliation, include by default
		return true
	}
}
