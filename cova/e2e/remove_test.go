package e2e_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/add"
	"github.com/MrPointer/agentcoven/cova/apply"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/remove"
)

// newRemoveDeps builds a remove.Deps with real Default* dependencies.
func newRemoveDeps(t *testing.T) remove.Deps {
	t.Helper()

	applyDeps := newApplyDeps(t, os.Getenv("HOME"))

	return remove.Deps{
		Logger:      applyDeps.Logger,
		FileSystem:  applyDeps.FileSystem,
		Locker:      applyDeps.Locker,
		BlockStore:  applyDeps.BlockStore,
		Dispatcher:  applyDeps.Dispatcher,
		EnvManager:  applyDeps.EnvManager,
		UserManager: applyDeps.UserManager,
	}
}

// addAndApply is a helper that runs add + apply for a repo with the given config.
func addAndApply(t *testing.T, repoURL string, cfg config.Config, tempHome string) {
	t.Helper()

	writeConfig(t, cfg)

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, repoURL, nil, "", false))

	applyDeps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), applyDeps, nil))
}

func TestRemove_RemovingSubscriptionShouldDeletePlacedFilesAndState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, _ := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")
	addAndApply(t, fileURL(repoDir), config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	}, tempHome)

	// Verify the file was placed.
	targetPath := claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md")
	require.FileExists(t, targetPath)

	// Remove the subscription.
	removeDeps := newRemoveDeps(t)
	require.NoError(t, remove.Run(t.Context(), removeDeps, []string{"acme-blocks"}))

	// File should be gone.
	require.NoFileExists(t, targetPath)

	// State should be empty.
	store := openBlockStore(t)
	records, err := store.QueryBySubscription(t.Context(), "acme-blocks")
	require.NoError(t, err)
	require.Empty(t, records)

	// Config should have no subscriptions.
	cfg := loadConfig(t)
	require.Empty(t, cfg.Subscriptions)
}

func TestRemove_RemovingSubscriptionShouldDeleteWorkspaceWhenNoOtherSubscriptionsUseIt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	_ = setupApplyEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	wsPath := resolveWorkspacePath(t, fileURL(repoDir))
	require.DirExists(t, wsPath)

	removeDeps := newRemoveDeps(t)
	require.NoError(t, remove.Run(t.Context(), removeDeps, []string{"acme-blocks"}))

	require.NoDirExists(t, wsPath)
}

func TestRemove_RemovingOneOfMultiCovenSubscriptionsShouldKeepWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	_ = setupApplyEnv(t)

	repoDir := createMultiCovenRepo(t, "acme", []string{"platform", "frontend"})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), []string{"platform", "frontend"}, "", false))

	wsPath := resolveWorkspacePath(t, fileURL(repoDir))
	require.DirExists(t, wsPath)

	removeDeps := newRemoveDeps(t)
	require.NoError(t, remove.Run(t.Context(), removeDeps, []string{"acme-platform"}))

	// Workspace should still exist — acme-frontend still uses it.
	require.DirExists(t, wsPath)

	// Only the removed subscription should be gone from config.
	cfg := loadConfig(t)
	require.Len(t, cfg.Subscriptions, 1)
	require.Equal(t, "acme-frontend", cfg.Subscriptions[0].Name)
}

func TestRemove_RemovingNonExistentSubscriptionShouldError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)
	writeConfig(t, config.Config{})

	removeDeps := newRemoveDeps(t)
	err := remove.Run(t.Context(), removeDeps, []string{"does-not-exist"})
	require.Error(t, err)
}

func TestRemove_RemovingWithSomeMissingShouldWarnAndContinue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	_ = setupApplyEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	removeDeps := newRemoveDeps(t)
	// "does-not-exist" is missing, but "acme-blocks" exists — should succeed.
	require.NoError(t, remove.Run(t.Context(), removeDeps, []string{"does-not-exist", "acme-blocks"}))

	cfg := loadConfig(t)
	require.Empty(t, cfg.Subscriptions)
}

func TestRemove_RemovingSubscriptionWithNoAppliedBlocksShouldStillCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	_ = setupApplyEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")

	// Add but don't apply — no state records, no placed files.
	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	wsPath := resolveWorkspacePath(t, fileURL(repoDir))
	require.DirExists(t, wsPath)

	removeDeps := newRemoveDeps(t)
	require.NoError(t, remove.Run(t.Context(), removeDeps, []string{"acme-blocks"}))

	// Config and workspace should still be cleaned up.
	cfg := loadConfig(t)
	require.Empty(t, cfg.Subscriptions)
	require.NoDirExists(t, wsPath)
}

func TestRemove_RemovingMultipleSubscriptionsShouldRemoveAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir1, _ := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")
	repoDir2 := createCovenRepoWithAgents(t, "other", "tools", "my-agent")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir1)},
			{Name: "other-tools", Repo: fileURL(repoDir2)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir1), nil, "", false))
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir2), nil, "", false))

	applyDeps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), applyDeps, nil))

	require.FileExists(t, claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md"))
	require.FileExists(t, claudeCodeAgentPath(tempHome, "my-agent", "agent.md"))

	removeDeps := newRemoveDeps(t)
	require.NoError(t, remove.Run(t.Context(), removeDeps, []string{"acme-blocks", "other-tools"}))

	require.NoFileExists(t, claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md"))
	require.NoFileExists(t, claudeCodeAgentPath(tempHome, "my-agent", "agent.md"))

	cfg := loadConfig(t)
	require.Empty(t, cfg.Subscriptions)

	store := openBlockStore(t)

	records1, err := store.QueryBySubscription(t.Context(), "acme-blocks")
	require.NoError(t, err)
	require.Empty(t, records1)

	records2, err := store.QueryBySubscription(t.Context(), "other-tools")
	require.NoError(t, err)
	require.Empty(t, records2)
}
