package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/remove"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

var removeCmd = &cobra.Command{
	Use:   "remove <name> [names...]",
	Short: "Remove one or more coven subscriptions",
	Long: `Remove one or more coven subscriptions and their placed files.

For each named subscription, remove placed files, state records, and the
config entry. The workspace directory is deleted if no remaining subscriptions
reference the same repository.

Missing subscriptions produce warnings; the command only errors if none of the
provided names exist in config.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.NewCliLogger(logger.Normal)
		defer log.Close()

		fs := utils.NewDefaultFileSystem(log)
		commander := utils.NewDefaultCommander(log)
		locker := utils.NewDefaultLocker()
		osManager := osmanager.NewDefaultOsManager(log, commander, fs)

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

		deps := remove.Deps{
			Logger:      log,
			FileSystem:  fs,
			Locker:      locker,
			BlockStore:  blockStore,
			Dispatcher:  dispatcher,
			EnvManager:  osManager,
			UserManager: osManager,
		}

		return remove.Run(context.Background(), deps, args)
	},
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	rootCmd.AddCommand(removeCmd)
}
