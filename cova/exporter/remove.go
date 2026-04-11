package exporter

import (
	"context"
	"fmt"

	"github.com/MrPointer/agentcoven/cova/config"
)

// Remove unregisters one or more exporters by name from the local configuration.
func Remove(ctx context.Context, deps Deps, names []string) error {
	configPath, err := config.DefaultPath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	results, err := config.RemoveAgents(ctx, deps.FileSystem, deps.Locker, configPath, names)
	if err != nil {
		return fmt.Errorf("removing exporters: %w", err)
	}

	for _, result := range results {
		if result.Removed {
			deps.Logger.Info("Removed exporter %q", result.Name)
		} else {
			deps.Logger.Warning("Exporter %q is not configured", result.Name)
		}
	}

	return nil
}
