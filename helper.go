package main

import (
	"fmt"
	"os"
	"runtime"
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
