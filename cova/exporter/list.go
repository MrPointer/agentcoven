package exporter

import (
	"context"
	"fmt"

	"github.com/MrPointer/agentcoven/cova/config"
)

// ExporterEntry describes a single exporter with its availability and configuration status.
type ExporterEntry struct {
	Name        string
	Description string
	BuiltIn     bool
	Configured  bool
}

// ListResult holds the outcome of listing available exporters.
type ListResult struct {
	BuiltIn  []ExporterEntry
	External []ExporterEntry
}

// List discovers all available exporters and reports which are configured.
func List(ctx context.Context, deps Deps) (ListResult, error) {
	available, err := deps.Dispatcher.ListAvailable(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("listing available exporters: %w", err)
	}

	if len(available) == 0 {
		return ListResult{}, nil
	}

	configPath, err := config.DefaultPath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return ListResult{}, fmt.Errorf("resolving config path: %w", err)
	}

	cfg, err := config.Load(deps.FileSystem, configPath)
	if err != nil {
		return ListResult{}, fmt.Errorf("loading config: %w", err)
	}

	configured := make(map[string]struct{}, len(cfg.Agents))
	for _, agent := range cfg.Agents {
		configured[agent] = struct{}{}
	}

	var result ListResult

	for _, info := range available {
		_, isCfg := configured[info.Name]

		entry := ExporterEntry{
			Name:        info.Name,
			Description: info.Description,
			BuiltIn:     info.BuiltIn,
			Configured:  isCfg,
		}

		if info.BuiltIn {
			result.BuiltIn = append(result.BuiltIn, entry)
		} else {
			result.External = append(result.External, entry)
		}
	}

	return result, nil
}
