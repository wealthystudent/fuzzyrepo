package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Config ConfigCmd `cmd:"" help:"Configure fuzzyrepo."`
}

type ConfigCmd struct {
	RepositoriesPath string `arg:"" optional:"" name:"path" help:"Path to repositories directory."`
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

// Runs the config command
func (c *ConfigCmd) Run() error {
	return c.setConfig()
}

func RunCLI(args []string) int {

	// Create parser
	k := kong.Must(&CLI,
		kong.Name("fuzzyrepo"),
		kong.Description("fuzzyrepo"),
		kong.UsageOnError(),
	)

	// Handle error
	ctx, err := k.Parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	if err := ctx.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
