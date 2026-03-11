package apply

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/MrPointer/agentcoven/cova/adapter"
	"github.com/MrPointer/agentcoven/cova/block"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/manifest"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

// Deps holds the injected service dependencies for the apply operation.
type Deps struct {
	Logger      logger.Logger
	FileSystem  utils.FileSystem
	Locker      utils.Locker
	Git         workspace.Git
	BlockStore  state.BlockStore
	Dispatcher  adapter.Dispatcher
	EnvManager  osmanager.EnvironmentManager
	UserManager osmanager.UserManager
}

// Run orchestrates the apply command: loads config, validates frameworks, and for each subscription
// creates a worktree, discovers blocks, resolves variants, invokes adapters, detects conflicts,
// copies files, records state, and cleans up orphans.
//
// If subscriptionNames is empty, all subscriptions in config are applied.
// If non-empty, only the named subscriptions are applied; an error is returned if a name is not found.
func Run(ctx context.Context, deps Deps, subscriptionNames []string) error {
	configPath, err := config.DefaultPath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	cfg, err := config.Load(deps.FileSystem, configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Frameworks) == 0 {
		return errors.New("no frameworks configured; add at least one framework to your config")
	}

	subs, err := selectSubscriptions(cfg, subscriptionNames)
	if err != nil {
		return err
	}

	basePath, err := workspace.DefaultBasePath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving workspace base path: %w", err)
	}

	for _, sub := range subs {
		if err := applySubscription(ctx, deps, cfg.Frameworks, basePath, sub); err != nil {
			deps.Logger.Warning("skipping subscription %q: %s", sub.Name, err)
		}
	}

	return nil
}

// selectSubscriptions returns the subscriptions to apply.
// If names is empty, all subscriptions are returned.
// If names is non-empty, only matching subscriptions are returned; an error is returned for any unknown name.
func selectSubscriptions(cfg config.Config, names []string) ([]config.Subscription, error) {
	if len(names) == 0 {
		return cfg.Subscriptions, nil
	}

	byName := make(map[string]config.Subscription, len(cfg.Subscriptions))
	for _, s := range cfg.Subscriptions {
		byName[s.Name] = s
	}

	result := make([]config.Subscription, 0, len(names))

	for _, name := range names {
		sub, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("subscription %q not found in config", name)
		}

		result = append(result, sub)
	}

	return result, nil
}

// subscriptionContext bundles the per-subscription workspace context passed into applyFramework.
type subscriptionContext struct {
	mf           *manifest.RootManifest
	blocks       map[string][]block.Block
	sub          config.Subscription
	worktreePath string
	covenRoot    string
}

// applySubscription handles the full apply flow for a single subscription.
// It returns an error if the subscription should be skipped (workspace missing, manifest error, etc.).
func applySubscription(
	ctx context.Context,
	deps Deps,
	frameworks []string,
	basePath string,
	sub config.Subscription,
) error {
	normalized, err := workspace.NormalizeURL(sub.Repo)
	if err != nil {
		return fmt.Errorf("normalizing repo URL %q: %w", sub.Repo, err)
	}

	repoDir := filepath.Join(basePath, normalized)

	exists, err := deps.FileSystem.PathExists(repoDir)
	if err != nil {
		return fmt.Errorf("checking workspace %q: %w", repoDir, err)
	}

	if !exists {
		return fmt.Errorf("workspace not found at %q (run 'cova add' first)", repoDir)
	}

	worktreePath, err := deps.Git.WorktreeAdd(ctx, repoDir, sub.Ref)
	if err != nil {
		return fmt.Errorf("creating worktree: %w", err)
	}

	mf, err := manifest.Parse(deps.FileSystem, worktreePath)
	if err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	covenRoot := worktreePath
	if !mf.IsSingleCoven() && sub.Path != "" {
		covenRoot = filepath.Join(worktreePath, sub.Path)
	}

	blocks, err := block.Discover(deps.FileSystem, covenRoot)
	if err != nil {
		return fmt.Errorf("discovering blocks: %w", err)
	}

	sc := subscriptionContext{
		sub:          sub,
		worktreePath: worktreePath,
		covenRoot:    covenRoot,
		mf:           mf,
		blocks:       blocks,
	}

	for _, framework := range frameworks {
		if err := applyFramework(ctx, deps, sc, framework); err != nil {
			deps.Logger.Warning("skipping framework %q for subscription %q: %s", framework, sub.Name, err)
		}
	}

	return nil
}

// applyFramework handles a single subscription+framework combination.
func applyFramework(ctx context.Context, deps Deps, sc subscriptionContext, framework string) error {
	resolved := buildResolvedBlocks(deps.FileSystem, sc.covenRoot, sc.worktreePath, sc.blocks, framework)

	if len(resolved) == 0 {
		return nil
	}

	req := &adapter.ApplyRequest{
		Blocks: resolved,
		Manifest: adapter.RequestManifest{
			Org:   sc.mf.Org,
			Coven: sc.mf.Covens[0],
		},
		Operation:    "apply",
		Subscription: sc.sub.Name,
		Workspace:    sc.worktreePath,
	}

	resp, err := deps.Dispatcher.Apply(ctx, framework, req)
	if err != nil {
		return fmt.Errorf("adapter failed: %w", err)
	}

	prevRecords, err := deps.BlockStore.QueryBySubscriptionFramework(ctx, sc.sub.Name, framework)
	if err != nil {
		return fmt.Errorf("querying existing state: %w", err)
	}

	prevPaths := make(map[string]struct{}, len(prevRecords))
	for _, r := range prevRecords {
		prevPaths[r.Path] = struct{}{}
	}

	newRecords := make([]state.Record, 0)
	appliedPaths := make(map[string]struct{})

	for _, result := range resp.Results {
		if result.Error != nil {
			deps.Logger.Warning(
				"block %q skipped by adapter (subscription %q, framework %q): %s",
				result.Name, sc.sub.Name, framework, *result.Error,
			)

			continue
		}

		for _, placement := range result.Placements {
			kind, err := checkConflict(ctx, deps.FileSystem, deps.BlockStore, placement.Path, sc.sub.Name, framework)
			if err != nil {
				deps.Logger.Warning(
					"conflict check failed for %q (subscription %q, framework %q): %s",
					placement.Path, sc.sub.Name, framework, err,
				)

				continue
			}

			switch kind {
			case conflictKindUserFile:
				deps.Logger.Warning(
					"conflict: %q already exists and is not managed by cova — skipping (subscription %q, framework %q)",
					placement.Path, sc.sub.Name, framework,
				)

				continue
			case conflictKindCrossSubscription:
				deps.Logger.Warning(
					"conflict: %q is managed by a different subscription — skipping (subscription %q, framework %q)",
					placement.Path, sc.sub.Name, framework,
				)

				continue
			default:
				// conflictKindNew and conflictKindUpdate: safe to write.
			}

			if err := deps.FileSystem.CreateDirectory(filepath.Dir(placement.Path)); err != nil {
				deps.Logger.Warning(
					"failed to create parent directory for %q (subscription %q, framework %q): %s",
					placement.Path, sc.sub.Name, framework, err,
				)

				continue
			}

			srcPath := filepath.Join(sc.worktreePath, placement.Source)

			if _, err := deps.FileSystem.CopyFile(srcPath, placement.Path); err != nil {
				deps.Logger.Warning(
					"failed to copy %q to %q (subscription %q, framework %q): %s",
					srcPath, placement.Path, sc.sub.Name, framework, err,
				)

				continue
			}

			newRecords = append(newRecords, state.Record{
				Path:         placement.Path,
				Subscription: sc.sub.Name,
				Source:       placement.Source,
				BlockType:    result.Name,
				Framework:    framework,
				Checksum:     "",
			})

			appliedPaths[placement.Path] = struct{}{}
		}
	}

	if len(newRecords) > 0 {
		if err := deps.BlockStore.RecordBatch(ctx, newRecords); err != nil {
			return fmt.Errorf("recording state: %w", err)
		}
	}

	if err := cleanupOrphans(ctx, deps, prevPaths, appliedPaths); err != nil {
		deps.Logger.Warning(
			"orphan cleanup failed (subscription %q, framework %q): %s",
			sc.sub.Name, framework, err,
		)
	}

	return nil
}

// buildResolvedBlocks resolves variants for all discovered blocks and returns the
// adapter request blocks map, keyed by block type.
// Sources are relative to the worktree root (not the coven root).
func buildResolvedBlocks(
	fs utils.FileSystem,
	covenRoot string,
	worktreePath string,
	blocks map[string][]block.Block,
	framework string,
) map[string][]adapter.RequestBlock {
	result := make(map[string][]adapter.RequestBlock)

	for blockType, blks := range blocks {
		for _, b := range blks {
			resolvedDir, include, err := block.ResolveVariant(fs, covenRoot, b.SourceDir, framework)
			if err != nil || !include {
				continue
			}

			// Convert from coven-root-relative to worktree-root-relative.
			worktreeRelSource := resolvedDir
			if covenRoot != worktreePath {
				rel, err := filepath.Rel(worktreePath, filepath.Join(covenRoot, resolvedDir))
				if err != nil {
					continue
				}

				worktreeRelSource = rel
			}

			result[blockType] = append(result[blockType], adapter.RequestBlock{
				Name:   b.Name,
				Source: worktreeRelSource,
			})
		}
	}

	return result
}

// cleanupOrphans deletes files and state records for paths that were previously tracked
// but are no longer present in the new applied set.
func cleanupOrphans(
	ctx context.Context,
	deps Deps,
	prevPaths map[string]struct{},
	appliedPaths map[string]struct{},
) error {
	var orphanPaths []string

	for p := range prevPaths {
		if _, stillPresent := appliedPaths[p]; !stillPresent {
			orphanPaths = append(orphanPaths, p)
		}
	}

	if len(orphanPaths) == 0 {
		return nil
	}

	for _, p := range orphanPaths {
		if err := deps.FileSystem.RemovePath(p); err != nil {
			deps.Logger.Warning("failed to remove orphan file %q: %s", p, err)
		}
	}

	if err := deps.BlockStore.DeleteByPaths(ctx, orphanPaths); err != nil {
		return fmt.Errorf("deleting orphan state records: %w", err)
	}

	return nil
}
