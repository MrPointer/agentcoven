package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

var exporterAddCmd = &cobra.Command{
	Use:   "add [name...]",
	Short: "Add one or more exporters to the configuration",
	Long: `Add one or more exporters to the local configuration.

If no names are provided, lists all available exporters (same as 'cova exporter list').
Use 'cova exporter list' to see available exporters and their descriptions.`,
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

		if len(args) == 0 {
			result, err := exporter.List(context.Background(), deps)
			if err != nil {
				return err
			}

			printListResult(result)

			return nil
		}

		return exporter.Add(context.Background(), deps, args)
	},
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	exporterCmd.AddCommand(exporterAddCmd)
}
