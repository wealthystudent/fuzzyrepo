package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
)

var CLI struct {
	Text string `arg:"" name:"text" help:"Arbitrary string input."`
}

func RunCLI(args []string) int {
	// Create parser
	k := kong.Must(&CLI,
		kong.Name("fuzzyrepo"),
		kong.Description("fuzzyrepo"),
		kong.UsageOnError(),
	)

	// Handle error
	_, err := k.Parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	// Expect exactly one string argument (./main thisisastring)
	if CLI.Text == "" {
		fmt.Fprintln(os.Stderr, "error: expected exactly one string argument")
		return 2
	}

	// TODO: Add functionallity for either opening a window with the search results, or open the samewindow with typing possibilities:
	return 0
}
