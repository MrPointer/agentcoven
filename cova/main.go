// Package main is the entry point for the cova CLI.
package main

import (
	"os"

	"github.com/MrPointer/agentcoven/cova/cmd"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
)

func main() {
	if err := cmd.Execute(); err != nil {
		logger.PrintStyled(os.Stderr, logger.ErrorStyle, "%s", err)
		os.Exit(1)
	}
}
