package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/adapter"
	"github.com/MrPointer/agentcoven/cova/apply"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

var applyCmd = &cobra.Command{
	Use:   "apply [names...]",
	Short: "Apply subscribed coven blocks to the local filesystem",
	Long: `Apply blocks from subscribed covens to the local filesystem.

If no names are given, all subscriptions are applied.
If one or more names are given, only those subscriptions are applied.`,
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

		dispatcher := adapter.NewDefaultDispatcher(osManager, commander, fs, homeDir)

		deps := apply.Deps{
			Logger:      log,
			FileSystem:  fs,
			Locker:      locker,
			Git:         git,
			BlockStore:  blockStore,
			Dispatcher:  dispatcher,
			EnvManager:  osManager,
			UserManager: osManager,
		}

		return apply.Run(context.Background(), deps, args)
	},
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	rootCmd.AddCommand(applyCmd)
}
