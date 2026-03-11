package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/add"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

var addRef string

var addCmd = &cobra.Command{
	Use:   "add <repo> [covens...]",
	Short: "Subscribe to a coven repository",
	Long: `Subscribe to one or more covens from a repository.

For single-coven repositories, extra coven arguments are silently ignored.
For multi-coven repositories, you must specify which covens to subscribe to.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log := logger.NewCliLogger(logger.Normal)
		defer log.Close()

		fs := utils.NewDefaultFileSystem(log)
		commander := utils.NewDefaultCommander(log)
		locker := utils.NewDefaultLocker()
		osManager := osmanager.NewDefaultOsManager(log, commander, fs)
		git := workspace.NewDefaultGit(commander, fs)

		deps := add.Deps{
			Logger:      log,
			FileSystem:  fs,
			Locker:      locker,
			Git:         git,
			EnvManager:  osManager,
			UserManager: osManager,
		}

		if err := add.Run(context.Background(), deps, args[0], args[1:], addRef); err != nil {
			log.Error("%s", err)
			os.Exit(1) //nolint:revive // deep-exit: Cobra Run (not RunE) requires inline exit per CLI skill.
		}
	},
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVar(&addRef, "ref", "", "pin subscription to a specific git ref (branch, tag, commit)")
}
