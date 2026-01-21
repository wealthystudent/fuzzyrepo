package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

func getOs() string {
	// get runtime OS
	return runtime.GOOS
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Could not find home directory:", err)
		return ""
	}
	fmt.Println("Your home directory is:", home)
	return home
}

// Makes a string exactly w characters long by cutting it down (adding “…” if needed) or padding it with spaces.
func padOrTrim(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if len(s) > w {
		if w <= 1 {
			return s[:w]
		}
		return s[:w-1] + "…"
	}
	return s + strings.Repeat(" ", w-len(s))
}

// Constrains v to the inclusive range [lo, hi] (returns lo if below, hi if above).
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func updateRepoJSON(config *Config) error {
	// ctx := context.Background()

	// Create GH Client
	// githubClient, err := getGithubClient(ctx)
	// if err != nil {
	// 	log.Fatal("Failed to create github client: ", err)
	// 	return err
	// }

	// Run logic for extracting from github and local and populate the json file

	return nil

}
