// Package main is the entry point for the cova CLI.
package main

import (
	"os"

	"github.com/MrPointer/agentcoven/cova/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
