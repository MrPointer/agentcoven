package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

var exporterRemoveCmd = &cobra.Command{
	Use:   "remove <name> [names...]",
	Short: "Remove one or more exporters from the configuration",
	Long:  `Remove one or more exporters from the local configuration.`,
	Args:  cobra.MinimumNArgs(1),
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

		dispatcher := exporter.NewDefaultDispatcher(osManager, commander, fs, homeDir)

		deps := exporter.Deps{
			Logger:      log,
			FileSystem:  fs,
			Locker:      locker,
			Dispatcher:  dispatcher,
			EnvManager:  osManager,
			UserManager: osManager,
		}

		return exporter.Remove(context.Background(), deps, args)
	},
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	exporterCmd.AddCommand(exporterRemoveCmd)
}
