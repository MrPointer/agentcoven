package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/status"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

var statusVerbose bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show subscriptions, applied blocks, and configured agents",
	Long: `Display the current state of cova: which covens are subscribed to, how many
blocks have been applied, and which agents are configured.

Use --verbose to see a per-subscription breakdown of applied blocks grouped
by block type.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.NewCliLogger(logger.Normal)
		defer log.Close()

		fs := utils.NewDefaultFileSystem(log)
		commander := utils.NewDefaultCommander(log)
		osManager := osmanager.NewDefaultOsManager(log, commander, fs)

		statePath, err := state.DefaultPath(osManager, osManager)
		if err != nil {
			return err
		}

		var blockStore state.BlockStore

		exists, err := fs.PathExists(statePath)
		if err != nil {
			return err
		}

		if exists {
			store, openErr := state.NewSQLiteBlockStore(fs, statePath)
			if openErr != nil {
				return openErr
			}

			defer store.Close()

			blockStore = store
		}

		deps := status.Deps{
			Logger:      log,
			FileSystem:  fs,
			BlockStore:  blockStore,
			EnvManager:  osManager,
			UserManager: osManager,
			Out:         os.Stdout,
		}

		return status.Run(context.Background(), deps, statusVerbose)
	},
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusVerbose, "verbose", "v", false, "Show per-subscription block breakdown")
}
