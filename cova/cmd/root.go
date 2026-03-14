// Package cmd contains the CLI commands for cova.
package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "cova",
	Short: "Reference CLI for the AgentCoven specification",
	Long: `cova is the reference implementation of the AgentCoven client specification.
It applies shared AI agent building blocks — skills, rules, agents — from coven
repositories to your local filesystem, translating them for your agent.`,
	SilenceUsage: true,
}

// Execute runs the root command and returns any error.
func Execute() error {
	return rootCmd.Execute()
}
