// Package add implements the orchestration logic for subscribing to coven repositories.
package add

import (
	"context"
	"fmt"
	"strings"

	"github.com/MrPointer/agentcoven/cova/apply"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/manifest"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

// Deps holds the injected service dependencies for the add operation.
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

// Run orchestrates the add command: ensures workspace, reads manifest, and upserts subscriptions.
// When apply is true, blocks for newly added or updated subscriptions are placed on disk immediately.
func Run(ctx context.Context, deps Deps, repoURL string, covenNames []string, ref string, applyBlocks bool) error {
	basePath, err := workspace.DefaultBasePath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving workspace base path: %w", err)
	}

	repoDir, err := workspace.Ensure(ctx, deps.Git, deps.FileSystem, basePath, repoURL, ref)
	if err != nil {
		return fmt.Errorf("ensuring workspace: %w", err)
	}

	mf, err := manifest.Parse(deps.FileSystem, repoDir)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	subs, err := BuildSubscriptions(mf, repoURL, ref, covenNames)
	if err != nil {
		return err
	}

	configPath, err := config.DefaultPath(deps.EnvManager, deps.UserManager)
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	var addedNames []string

	for _, sub := range subs {
		result, err := config.UpsertSubscription(ctx, deps.FileSystem, deps.Locker, configPath, sub)
		if err != nil {
			return fmt.Errorf("upserting subscription %q: %w", sub.Name, err)
		}

		LogUpsertResult(deps.Logger, sub.Name, result)

		if result == config.UpsertAdded || result == config.UpsertUpdated {
			addedNames = append(addedNames, sub.Name)
		}
	}

	if applyBlocks && len(addedNames) > 0 {
		applyDeps := apply.Deps{
			Logger:      deps.Logger,
			FileSystem:  deps.FileSystem,
			Locker:      deps.Locker,
			Git:         deps.Git,
			BlockStore:  deps.BlockStore,
			Dispatcher:  deps.Dispatcher,
			EnvManager:  deps.EnvManager,
			UserManager: deps.UserManager,
		}

		if err := apply.Run(ctx, applyDeps, addedNames); err != nil {
			return fmt.Errorf("subscriptions added successfully, but apply failed — run `cova apply` to retry: %w", err)
		}
	}

	return nil
}

// BuildSubscriptions determines which subscriptions to create based on the manifest layout.
func BuildSubscriptions(
	mf *manifest.RootManifest,
	repoURL string,
	ref string,
	covenNames []string,
) ([]config.Subscription, error) {
	if mf.IsSingleCoven() {
		return buildSingleCovenSubscriptions(mf, repoURL, ref), nil
	}

	return buildMultiCovenSubscriptions(mf, repoURL, ref, covenNames)
}

func buildSingleCovenSubscriptions(
	mf *manifest.RootManifest,
	repoURL string,
	ref string,
) []config.Subscription {
	name := mf.Org + "-" + mf.Covens[0]

	return []config.Subscription{
		{
			Name: name,
			Repo: repoURL,
			Ref:  ref,
		},
	}
}

func buildMultiCovenSubscriptions(
	mf *manifest.RootManifest,
	repoURL string,
	ref string,
	covenNames []string,
) ([]config.Subscription, error) {
	if len(covenNames) == 0 {
		return nil, fmt.Errorf(
			"repository contains multiple covens; specify one or more: %s",
			strings.Join(mf.Covens, ", "),
		)
	}

	available := make(map[string]struct{}, len(mf.Covens))
	for _, c := range mf.Covens {
		available[c] = struct{}{}
	}

	subs := make([]config.Subscription, 0, len(covenNames))

	for _, cn := range covenNames {
		if _, ok := available[cn]; !ok {
			return nil, fmt.Errorf(
				"coven %q not found in manifest; available: %s",
				cn, strings.Join(mf.Covens, ", "),
			)
		}

		subs = append(subs, config.Subscription{
			Name: mf.Org + "-" + cn,
			Repo: repoURL,
			Path: "covens/" + cn,
			Ref:  ref,
		})
	}

	return subs, nil
}

// LogUpsertResult logs the outcome of a subscription upsert operation.
func LogUpsertResult(log logger.Logger, name string, result config.UpsertResult) {
	switch result {
	case config.UpsertAdded:
		log.Success("added subscription %s", name)
	case config.UpsertUpdated:
		log.Info("updated subscription %s", name)
	case config.UpsertNoOp:
		log.Info("subscription %s already up to date", name)
	default:
		log.Warning("unexpected upsert result for subscription %s", name)
	}
}
