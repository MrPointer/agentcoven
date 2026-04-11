package exporter

import (
	"context"
	"fmt"

	"github.com/MrPointer/agentcoven/cova/config"
)

// Add registers one or more exporters by name in the local configuration.
func Add(ctx context.Context, deps Deps, names []string) error {
	configPath, err := config.DefaultPath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	results, err := config.AddAgents(ctx, deps.FileSystem, deps.Locker, configPath, names)
	if err != nil {
		return fmt.Errorf("adding exporters: %w", err)
	}

	for _, result := range results {
		if result.Added {
			deps.Logger.Info("Added exporter %q", result.Name)
		} else {
			deps.Logger.Info("Exporter %q is already configured", result.Name)
		}
	}

	return nil
}
