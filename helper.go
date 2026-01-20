package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".fuzzyrepo.conf"), nil
}

func stripTilde(path string) {

}

func StripTilde(path string) (string, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", fmt.Errorf("empty path")
	}

	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}

		rest := strings.TrimPrefix(p, "~")
		rest = strings.TrimPrefix(rest, "/")
		rest = strings.TrimPrefix(rest, `\`)

		if rest == "" {
			return home, nil
		}
		return filepath.Join(home, rest), nil
	}

	return p, nil
}
