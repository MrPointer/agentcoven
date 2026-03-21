package e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/add"
	"github.com/MrPointer/agentcoven/cova/apply"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/state"
)

func TestApply_ApplyingSubscriptionWithSkillsShouldPlaceFilesAtCorrectPaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, skillContent := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, nil))

	targetPath := claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md")
	require.FileExists(t, targetPath)

	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, skillContent, string(content))
}

func TestApply_ApplyingSubscriptionWithAgentsShouldPlaceFilesAtCorrectPaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir := createCovenRepoWithAgents(t, "acme", "blocks", "my-agent")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, nil))

	targetPath := claudeCodeAgentPath(tempHome, "my-agent", "agent.md")
	require.FileExists(t, targetPath)
}

func TestApply_ApplyingMixedBlockTypesShouldPlaceSupportedTypesAndSkipUnsupported(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir := createCovenRepoWithMixedBlocks(t, "acme", "blocks")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, nil))

	// Supported types must be placed.
	require.FileExists(t, claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md"))
	require.FileExists(t, claudeCodeAgentPath(tempHome, "my-agent", "agent.md"))

	// Unsupported type (rules) must not produce a file.
	rulesPath := filepath.Join(tempHome, ".claude", "rules", "my-rule", "RULE.md")
	require.NoFileExists(t, rulesPath)
}

func TestApply_ApplyingWithNoAgentsConfiguredShouldSucceedWithNoOp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, _ := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: nil,
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	err := apply.Run(t.Context(), deps, nil)
	require.NoError(t, err)
}

func TestApply_ApplyingNamedSubscriptionShouldOnlyPlaceThatSubscriptionsBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repo1, _ := createCovenRepoWithSkills(t, "acme", "alpha", "alpha-skill")
	repo2, _ := createCovenRepoWithSkills(t, "acme", "beta", "beta-skill")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-alpha", Repo: fileURL(repo1)},
			{Name: "acme-beta", Repo: fileURL(repo2)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repo1), nil, "", false))
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repo2), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, []string{"acme-alpha"}))

	// Only alpha subscription blocks should be placed.
	require.FileExists(t, claudeCodeSkillPath(tempHome, "alpha-skill", "SKILL.md"))
	require.NoFileExists(t, claudeCodeSkillPath(tempHome, "beta-skill", "SKILL.md"))
}

func TestApply_ApplyingUnknownSubscriptionNameShouldReturnError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, _ := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	err := apply.Run(t.Context(), deps, []string{"nonexistent-sub"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent-sub")
}

func TestApply_ApplyingShouldRecordStateForPlacedFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, _ := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, nil))

	targetPath := claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md")

	store := openBlockStore(t)
	rec, err := store.QueryByPath(t.Context(), targetPath)
	require.NoError(t, err)
	require.Equal(t, targetPath, rec.Path)
	require.Equal(t, "acme-blocks", rec.Subscription)
	require.Equal(t, "skills", rec.BlockType)
	require.Equal(t, "claude-code", rec.Agent)
}

func TestApply_ReApplyingShouldBeIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, skillContent := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, nil))
	require.NoError(t, apply.Run(t.Context(), deps, nil))

	targetPath := claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md")
	require.FileExists(t, targetPath)

	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, skillContent, string(content))

	store := openBlockStore(t)
	records, err := store.QueryBySubscriptionAgent(t.Context(), "acme-blocks", "claude-code")
	require.NoError(t, err)
	require.Len(t, records, 1, "re-apply must not duplicate state records")
}

func TestApply_RemovingBlockFromRepoAndReApplyingShouldDeleteOrphanedFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	// Create a repo with two skill blocks.
	repoDir := createGitRepo(t, map[string]string{
		"manifest.yaml":              singleCovenManifest("acme", "blocks"),
		"skills/keep-skill/SKILL.md": "keep content",
		"skills/drop-skill/SKILL.md": "drop content",
	})

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, nil))

	// Both files should be placed after the first apply.
	keepPath := claudeCodeSkillPath(tempHome, "keep-skill", "SKILL.md")
	dropPath := claudeCodeSkillPath(tempHome, "drop-skill", "SKILL.md")

	require.FileExists(t, keepPath)
	require.FileExists(t, dropPath)

	// Remove drop-skill from the source repo and commit the change.
	// Then advance the workspace to the new commit (workspace.Ensure fetches but does not
	// advance HEAD, so we do it manually to simulate a fetch+reset flow).
	// (workspace.Ensure fetches but does not advance HEAD; we do it manually here to
	// simulate what a fetch+reset flow would produce.)
	require.NoError(t, os.RemoveAll(filepath.Join(repoDir, "skills", "drop-skill")))
	gitCmd(t, repoDir, "add", ".")
	gitCmd(t, repoDir, "commit", "-m", "remove drop-skill")

	workspacePath := resolveWorkspacePath(t, fileURL(repoDir))
	gitCmd(t, workspacePath, "fetch", "--all")
	gitCmd(t, workspacePath, "reset", "--hard", "FETCH_HEAD")

	require.NoError(t, apply.Run(t.Context(), deps, nil))

	// drop-skill file must be deleted; keep-skill must still exist.
	require.FileExists(t, keepPath)
	require.NoFileExists(t, dropPath)

	store := openBlockStore(t)
	_, err := store.QueryByPath(t.Context(), dropPath)
	require.ErrorIs(t, err, state.ErrNotFound)
}

func TestApply_UserFileConflictShouldNotOverwriteExistingUnmanagedFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, _ := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")

	writeConfig(t, config.Config{
		Subscriptions: []config.Subscription{
			{Name: "acme-blocks", Repo: fileURL(repoDir)},
		},
		Agents: []string{"claude-code"},
	})

	addDeps := newDeps(t)
	require.NoError(t, add.Run(t.Context(), addDeps, fileURL(repoDir), nil, "", false))

	// Pre-create the target file as a user file (not tracked by cova).
	targetPath := claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(targetPath), 0o755))
	require.NoError(t, os.WriteFile(targetPath, []byte("user content"), 0o644))

	deps := newApplyDeps(t, tempHome)
	require.NoError(t, apply.Run(t.Context(), deps, nil))

	// The user file must not have been overwritten.
	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, "user content", string(content))
}

func TestAdd_AddingWithApplyBlocksTrueShouldPlaceBlocksImmediately(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	tempHome := setupApplyEnv(t)

	repoDir, skillContent := createCovenRepoWithSkills(t, "acme", "blocks", "my-skill")

	// Write a config with agents already configured so apply can succeed.
	writeConfig(t, config.Config{
		Agents: []string{"claude-code"},
	})

	deps := newDeps(t)

	// Extend add.Deps with the apply-capable fields.
	addDepsWithApply := addDepsWithApplySupport(t, deps, tempHome)

	err := add.Run(t.Context(), addDepsWithApply, fileURL(repoDir), nil, "", true)
	require.NoError(t, err)

	// Blocks must be placed immediately without calling apply.Run separately.
	targetPath := claudeCodeSkillPath(tempHome, "my-skill", "SKILL.md")
	require.FileExists(t, targetPath)

	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, skillContent, string(content))
}

// addDepsWithApplySupport creates an add.Deps that includes BlockStore and Dispatcher
// so that applyBlocks=true works end-to-end.
func addDepsWithApplySupport(t *testing.T, base add.Deps, homeDir string) add.Deps {
	t.Helper()

	applyDeps := newApplyDeps(t, homeDir)

	return add.Deps{
		Logger:      base.Logger,
		FileSystem:  base.FileSystem,
		Locker:      base.Locker,
		Git:         base.Git,
		BlockStore:  applyDeps.BlockStore,
		Dispatcher:  applyDeps.Dispatcher,
		EnvManager:  base.EnvManager,
		UserManager: base.UserManager,
	}
}
