package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

var exporterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available exporters",
	Long:  `List all available exporters with descriptions and configured status.`,
	Args:  cobra.NoArgs,
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

		result, err := exporter.List(context.Background(), deps)
		if err != nil {
			return err
		}

		printListResult(result)

		return nil
	},
}

func printListResult(result exporter.ListResult) {
	if len(result.BuiltIn) == 0 && len(result.External) == 0 {
		fmt.Fprintln(os.Stdout, "No exporters available.")
		return
	}

	if len(result.BuiltIn) > 0 {
		fmt.Fprintln(os.Stdout, "Built-in:")
		printExporterEntries(result.BuiltIn)
	}

	if len(result.External) > 0 {
		if len(result.BuiltIn) > 0 {
			fmt.Fprintln(os.Stdout)
		}

		fmt.Fprintln(os.Stdout, "External:")
		printExporterEntries(result.External)
	}
}

func printExporterEntries(entries []exporter.ExporterEntry) {
	for _, e := range entries {
		marker := ""
		if e.Configured {
			marker = " [configured]"
		}

		fmt.Fprintf(os.Stdout, "  - %s: %s%s\n", e.Name, e.Description, marker)
	}
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	exporterCmd.AddCommand(exporterListCmd)
}
