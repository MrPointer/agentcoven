package apply

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/MrPointer/agentcoven/cova/block"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/exporter"
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
	Dispatcher  exporter.Dispatcher
	EnvManager  osmanager.EnvironmentManager
	UserManager osmanager.UserManager
}

// Run orchestrates the apply command: loads config, validates agents, and for each subscription
// creates a worktree, discovers blocks, resolves variants, invokes exporters, detects conflicts,
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

	if len(cfg.Agents) == 0 {
		deps.Logger.Warning("no agents configured — skipping application (add agents to your config)")

		return nil
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
		if err := applySubscription(ctx, deps, cfg.Agents, basePath, sub); err != nil {
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

// subscriptionContext bundles the per-subscription workspace context passed into applyAgent.
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
	agents []string,
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

	for _, agent := range agents {
		if err := applyAgent(ctx, deps, sc, agent); err != nil {
			deps.Logger.Warning("skipping agent %q for subscription %q: %s", agent, sub.Name, err)
		}
	}

	return nil
}

// applyAgent handles a single subscription+agent combination.
func applyAgent(ctx context.Context, deps Deps, sc subscriptionContext, agent string) error {
	resolved := buildResolvedBlocks(deps.FileSystem, sc.covenRoot, sc.worktreePath, sc.blocks, agent)

	if len(resolved) == 0 {
		return nil
	}

	req := &exporter.ApplyRequest{
		Blocks: resolved,
		Manifest: exporter.RequestManifest{
			Org:   sc.mf.Org,
			Coven: sc.mf.Covens[0],
		},
		Operation:    "apply",
		Subscription: sc.sub.Name,
		Workspace:    sc.worktreePath,
	}

	resp, err := deps.Dispatcher.Apply(ctx, agent, req)
	if err != nil {
		return fmt.Errorf("exporter failed: %w", err)
	}

	prevRecords, err := deps.BlockStore.QueryBySubscriptionAgent(ctx, sc.sub.Name, agent)
	if err != nil {
		return fmt.Errorf("querying existing state: %w", err)
	}

	prevPaths := make(map[string]struct{}, len(prevRecords))
	for _, r := range prevRecords {
		prevPaths[r.Path] = struct{}{}
	}

	// Build an ordered list of (blockType, blockName) pairs from the request.
	// This preserves the order and allows us to match response results with their types.
	blockTypeNamePairs := buildBlockTypeNamePairs(resolved)

	newRecords := make([]state.Record, 0)
	appliedPaths := make(map[string]struct{})

	for i, result := range resp.Results {
		if result.Error != nil {
			deps.Logger.Warning(
				"block %q skipped by exporter (subscription %q, agent %q): %s",
				result.Name, sc.sub.Name, agent, *result.Error,
			)

			continue
		}

		// Get the block type from the ordered pairs (exporter protocol guarantees result order matches input order).
		blockType := ""
		if i < len(blockTypeNamePairs) {
			blockType = blockTypeNamePairs[i].blockType
		}

		for _, placement := range result.Placements {
			kind, err := checkConflict(ctx, deps.FileSystem, deps.BlockStore, placement.Path, sc.sub.Name, agent)
			if err != nil {
				deps.Logger.Warning(
					"conflict check failed for %q (subscription %q, agent %q): %s",
					placement.Path, sc.sub.Name, agent, err,
				)

				continue
			}

			switch kind {
			case conflictKindUserFile:
				deps.Logger.Warning(
					"conflict: %q already exists and is not managed by cova — skipping (subscription %q, agent %q)",
					placement.Path, sc.sub.Name, agent,
				)

				continue
			case conflictKindCrossSubscription:
				deps.Logger.Warning(
					"conflict: %q is managed by a different subscription — skipping (subscription %q, agent %q)",
					placement.Path, sc.sub.Name, agent,
				)

				continue
			default:
				// conflictKindNew and conflictKindUpdate: safe to write.
			}

			if err := deps.FileSystem.CreateDirectory(filepath.Dir(placement.Path)); err != nil {
				deps.Logger.Warning(
					"failed to create parent directory for %q (subscription %q, agent %q): %s",
					placement.Path, sc.sub.Name, agent, err,
				)

				continue
			}

			srcPath := filepath.Join(sc.worktreePath, placement.Source)

			if _, err := deps.FileSystem.CopyFile(srcPath, placement.Path); err != nil {
				deps.Logger.Warning(
					"failed to copy %q to %q (subscription %q, agent %q): %s",
					srcPath, placement.Path, sc.sub.Name, agent, err,
				)

				continue
			}

			newRecords = append(newRecords, state.Record{
				Path:         placement.Path,
				Subscription: sc.sub.Name,
				Source:       placement.Source,
				BlockType:    blockType,
				Agent:        agent,
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
			"orphan cleanup failed (subscription %q, agent %q): %s",
			sc.sub.Name, agent, err,
		)
	}

	return nil
}

// blockTypeNamePair represents a block type and name pair for ordering.
type blockTypeNamePair struct {
	blockType string
	blockName string
}

// buildBlockTypeNamePairs builds an ordered list of (blockType, blockName) pairs from the resolved blocks map.
// The order is deterministic and matches the order of results from the exporter (which uses the same order).
func buildBlockTypeNamePairs(resolved map[string][]exporter.RequestBlock) []blockTypeNamePair {
	var pairs []blockTypeNamePair

	blockTypes := slices.Sorted(maps.Keys(resolved))

	for _, blockType := range blockTypes {
		for _, rb := range resolved[blockType] {
			pairs = append(pairs, blockTypeNamePair{
				blockType: blockType,
				blockName: rb.Name,
			})
		}
	}

	return pairs
}

// buildResolvedBlocks resolves variants for all discovered blocks and returns the
// exporter request blocks map, keyed by block type.
// Sources are relative to the worktree root (not the coven root).
func buildResolvedBlocks(
	fs utils.FileSystem,
	covenRoot string,
	worktreePath string,
	blocks map[string][]block.Block,
	agent string,
) map[string][]exporter.RequestBlock {
	result := make(map[string][]exporter.RequestBlock)

	for blockType, blks := range blocks {
		for _, b := range blks {
			resolvedDir, include, err := block.ResolveVariant(fs, covenRoot, b.SourceDir, agent)
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

			result[blockType] = append(result[blockType], exporter.RequestBlock{
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
