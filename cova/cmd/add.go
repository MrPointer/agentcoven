package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/add"
	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/state"
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
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.NewCliLogger(logger.Normal)
		defer log.Close()

		fs := utils.NewDefaultFileSystem(log)
		commander := utils.NewDefaultCommander(log)
		locker := utils.NewDefaultLocker()
		osManager := osmanager.NewDefaultOsManager(log, commander, fs)
		git := workspace.NewDefaultGit(commander, fs)

		homeDir, err := osManager.GetHomeDir()
		if err != nil {
			return err
		}

		statePath, err := state.DefaultPath(osManager, osManager)
		if err != nil {
			return err
		}

		blockStore, err := state.NewSQLiteBlockStore(fs, statePath)
		if err != nil {
			return err
		}

		defer blockStore.Close()

		dispatcher := exporter.NewDefaultDispatcher(osManager, commander, fs, homeDir)

		deps := add.Deps{
			Logger:      log,
			FileSystem:  fs,
			Locker:      locker,
			Git:         git,
			BlockStore:  blockStore,
			Dispatcher:  dispatcher,
			EnvManager:  osManager,
			UserManager: osManager,
		}

		return add.Run(context.Background(), deps, args[0], args[1:], addRef, true)
	},
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVar(&addRef, "ref", "", "pin subscription to a specific git ref (branch, tag, commit)")
}
