package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// Make sure the fuzzyrepo configuration file exists
func (c *ConfigCmd) setConfig() error {
	// Check config path
	cfgPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Println("Config file already exists:", cfgPath)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	// Determine repo path. use CLI arg if provided. otherwise prompt
	path := strings.TrimSpace(c.RepositoriesPath)
	if path == "" {
		fmt.Print("Insert path to local repositories directory: ")
		in := bufio.NewReader(os.Stdin)
		line, err := in.ReadString('\n')
		if err != nil {
			return err
		}
		path = strings.TrimSpace(line)
	}

	if path == "" {
		return fmt.Errorf("empty path provided")
	}

	//convert tilde to homedir
	repoPath, err := StripTilde(path)
	if err != nil {
		return err
	}

	configFileContents := fmt.Sprintf("repository-path: %s\n", repoPath)
	if err := os.WriteFile(cfgPath, []byte(configFileContents), 0644); err != nil {
		return err
	}

	fmt.Println("Wrote config:", cfgPath)
	return nil
}

func getOs() string {
	// get runtime OS
	return runtime.GOOS
}
