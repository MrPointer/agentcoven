package cmd

import "github.com/spf13/cobra"

var exporterCmd = &cobra.Command{
	Use:   "exporter",
	Short: "Manage exporters",
	Long:  `Manage which exporters are configured for block placement.`,
}

//nolint:gochecknoinits // Cobra requires an init function to set up the command structure.
func init() {
	rootCmd.AddCommand(exporterCmd)
}
