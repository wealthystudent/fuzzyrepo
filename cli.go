package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Config ConfigCmd `cmd:"" help:"Configure fuzzyrepo."`
}

type ConfigCmd struct {
	RepositoriesPath string `arg:"" optional:"" name:"path" help:"Path to repositories directory."`
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
