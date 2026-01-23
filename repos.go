package main

import (
	"bufio"
	"os"
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
