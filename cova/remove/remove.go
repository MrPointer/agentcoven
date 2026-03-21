// Package remove implements the orchestration logic for removing coven subscriptions.
package remove

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/manifest"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

// Deps holds the injected service dependencies for the remove operation.
type Deps struct {
	Logger      logger.Logger
	FileSystem  utils.FileSystem
	Locker      utils.Locker
	BlockStore  state.BlockStore
	Dispatcher  exporter.Dispatcher
	EnvManager  osmanager.EnvironmentManager
	UserManager osmanager.UserManager
}

// Run orchestrates the remove command: for each named subscription, removes placed files,
// state records, config entry, and workspace directory (if no other subscriptions reference it).
//
// Missing subscriptions produce warnings; the command only errors if none of the provided names
// exist in config.
func Run(ctx context.Context, deps Deps, names []string) error {
	configPath, err := config.DefaultPath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	basePath, err := workspace.DefaultBasePath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving workspace base path: %w", err)
	}

	cfg, err := config.Load(deps.FileSystem, configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	byName := make(map[string]config.Subscription, len(cfg.Subscriptions))
	for _, s := range cfg.Subscriptions {
		byName[s.Name] = s
	}

	var foundCount int

	for _, name := range names {
		sub, ok := byName[name]
		if !ok {
			deps.Logger.Warning("subscription %q not found in config — skipping", name)

			continue
		}

		foundCount++

		if err := removeSubscription(ctx, deps, configPath, basePath, sub); err != nil {
			deps.Logger.Warning("failed to remove subscription %q: %s", name, err)
		}
	}

	if foundCount == 0 {
		return errors.New("none of the provided subscription names exist in config")
	}

	return nil
}

// removeSubscription handles the full remove flow for a single subscription.
func removeSubscription(
	ctx context.Context,
	deps Deps,
	configPath string,
	basePath string,
	sub config.Subscription,
) error {
	records, err := deps.BlockStore.QueryBySubscription(ctx, sub.Name)
	if err != nil {
		return fmt.Errorf("querying state records: %w", err)
	}

	if len(records) > 0 {
		if notifyErr := notifyExportersAndDeleteFiles(ctx, deps, basePath, sub, records); notifyErr != nil {
			return notifyErr
		}

		if deleteErr := deps.BlockStore.DeleteBySubscription(ctx, sub.Name); deleteErr != nil {
			return fmt.Errorf("deleting state records: %w", deleteErr)
		}
	}

	removed, err := config.RemoveSubscription(ctx, deps.FileSystem, deps.Locker, configPath, sub.Name)
	if err != nil {
		return fmt.Errorf("removing subscription from config: %w", err)
	}

	if !removed {
		deps.Logger.Warning("subscription %q was not found during config removal (concurrent modification?)", sub.Name)
	}

	if err := cleanupWorkspaceIfUnused(ctx, deps, configPath, basePath, sub.Repo); err != nil {
		deps.Logger.Warning("workspace cleanup failed for subscription %q: %s", sub.Name, err)
	}

	deps.Logger.Success("removed subscription %s", sub.Name)

	return nil
}

// notifyExportersAndDeleteFiles notifies exporters of the removal and then deletes the placed files.
func notifyExportersAndDeleteFiles(
	ctx context.Context,
	deps Deps,
	basePath string,
	sub config.Subscription,
	records []state.Record,
) error {
	mf, workspaceDir, ok := tryLoadManifest(deps, basePath, sub)

	if ok {
		notifyExporters(ctx, deps, sub, workspaceDir, mf, records)
	}

	for _, r := range records {
		if err := deps.FileSystem.RemovePath(r.Path); err != nil {
			deps.Logger.Warning("failed to delete file %q (subscription %q): %s", r.Path, sub.Name, err)
		}
	}

	return nil
}

// tryLoadManifest attempts to resolve the workspace directory and parse the manifest.
// Returns the manifest, workspace directory, and whether the manifest was successfully loaded.
func tryLoadManifest(
	deps Deps,
	basePath string,
	sub config.Subscription,
) (*manifest.RootManifest, string, bool) {
	normalized, err := workspace.NormalizeURL(sub.Repo)
	if err != nil {
		deps.Logger.Warning("cannot normalize repo URL %q — skipping exporter notification: %s", sub.Repo, err)

		return nil, "", false
	}

	workspaceDir := filepath.Join(basePath, normalized)

	exists, err := deps.FileSystem.PathExists(workspaceDir)
	if err != nil {
		deps.Logger.Warning(
			"cannot check workspace %q — skipping exporter notification: %s",
			workspaceDir, err,
		)

		return nil, "", false
	}

	if !exists {
		deps.Logger.Warning(
			"workspace %q not found — skipping exporter notification (already deleted?)",
			workspaceDir,
		)

		return nil, "", false
	}

	mf, err := manifest.Parse(deps.FileSystem, workspaceDir)
	if err != nil {
		deps.Logger.Warning(
			"cannot parse manifest in %q — skipping exporter notification: %s",
			workspaceDir, err,
		)

		return nil, "", false
	}

	return mf, workspaceDir, true
}

// notifyExporters sends a remove request to each agent's exporter.
// Errors are logged as warnings; failures do not abort cleanup.
func notifyExporters(
	ctx context.Context,
	deps Deps,
	sub config.Subscription,
	workspaceDir string,
	mf *manifest.RootManifest,
	records []state.Record,
) {
	coven := resolveCoven(mf, sub)

	byAgent := groupRecordsByAgent(records)

	for agent, agentRecords := range byAgent {
		req := buildRemoveRequest(sub, coven, mf.Org, agent, agentRecords)

		_, err := deps.Dispatcher.Remove(ctx, agent, req)
		if err != nil {
			deps.Logger.Warning(
				"exporter notification failed for agent %q (subscription %q): %s",
				agent, sub.Name, err,
			)
		}
	}
}

// resolveCoven returns the coven name to use in the remove request.
// For single-coven repos, it returns the first (and only) coven name.
// For multi-coven repos, the coven name is derived from the subscription's Path field.
func resolveCoven(mf *manifest.RootManifest, sub config.Subscription) string {
	if mf.IsSingleCoven() || sub.Path == "" {
		return mf.Covens[0]
	}

	return filepath.Base(sub.Path)
}

// groupRecordsByAgent groups records by their agent field.
func groupRecordsByAgent(records []state.Record) map[string][]state.Record {
	byAgent := make(map[string][]state.Record)

	for _, r := range records {
		byAgent[r.Agent] = append(byAgent[r.Agent], r)
	}

	return byAgent
}

// buildRemoveRequest constructs a RemoveRequest for a specific agent from its records.
func buildRemoveRequest(
	sub config.Subscription,
	coven string,
	org string,
	agent string,
	records []state.Record,
) *exporter.RemoveRequest {
	// Group by block type, then accumulate paths per block name.
	type blockKey struct {
		blockType string
		blockName string
	}

	byBlock := make(map[blockKey][]string)

	for _, r := range records {
		key := blockKey{blockType: r.BlockType, blockName: filepath.Base(r.Source)}
		byBlock[key] = append(byBlock[key], r.Path)
	}

	blocks := make(map[string][]exporter.RemoveRequestBlock)

	for key, paths := range byBlock {
		blocks[key.blockType] = append(blocks[key.blockType], exporter.RemoveRequestBlock{
			Name:  key.blockName,
			Paths: paths,
		})
	}

	return &exporter.RemoveRequest{
		Blocks: blocks,
		Manifest: exporter.RequestManifest{
			Org:   org,
			Coven: coven,
		},
		Operation:    "remove",
		Subscription: sub.Name,
	}
}

// cleanupWorkspaceIfUnused deletes the workspace directory when no remaining subscriptions
// reference the same normalized repo URL.
func cleanupWorkspaceIfUnused(
	ctx context.Context,
	deps Deps,
	configPath string,
	basePath string,
	repoURL string,
) error {
	// Reload config after removal to get the current subscription list.
	cfg, err := config.Load(deps.FileSystem, configPath)
	if err != nil {
		return fmt.Errorf("reloading config: %w", err)
	}

	normalizedRemoved, err := workspace.NormalizeURL(repoURL)
	if err != nil {
		return fmt.Errorf("normalizing removed repo URL: %w", err)
	}

	for _, remaining := range cfg.Subscriptions {
		normalizedRemaining, normalizeErr := workspace.NormalizeURL(remaining.Repo)
		if normalizeErr != nil {
			// Can't compare — assume the workspace is still needed.
			deps.Logger.Warning(
				"cannot normalize remaining subscription repo URL %q — keeping workspace: %s",
				remaining.Repo, normalizeErr,
			)

			return nil
		}

		if normalizedRemaining == normalizedRemoved {
			// Another subscription still uses this workspace.
			return nil
		}
	}

	workspaceDir := filepath.Join(basePath, normalizedRemoved)

	if err := deps.FileSystem.RemovePath(workspaceDir); err != nil {
		return fmt.Errorf("deleting workspace %q: %w", workspaceDir, err)
	}

	return nil
}
