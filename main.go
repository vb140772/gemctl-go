package main

import (
	"fmt"
	"os"

	"github.com/vb140772/gemctl-go/internal/cli"
)

var version = "dev" // This will be replaced by ldflags during build

func main() {
	rootCmd := cli.NewRootCommand(version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
