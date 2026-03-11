package e2e_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/add"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

// newDeps builds an add.Deps with real Default* dependencies.
func newDeps(t *testing.T) add.Deps {
	t.Helper()

	log := logger.NoopLogger{}
	fs := utils.NewDefaultFileSystem(log)
	cmdr := utils.NewDefaultCommander(log)
	osMgr := osmanager.NewDefaultOsManager(log, cmdr, fs)

	return add.Deps{
		Logger:      log,
		FileSystem:  fs,
		Locker:      utils.NewDefaultLocker(),
		Git:         workspace.NewDefaultGit(cmdr, fs),
		EnvManager:  osMgr,
		UserManager: osMgr,
	}
}

// loadConfig loads and returns the config from the XDG_CONFIG_HOME set for the test.
func loadConfig(t *testing.T) config.Config {
	t.Helper()

	log := logger.NoopLogger{}
	fs := utils.NewDefaultFileSystem(log)
	cmdr := utils.NewDefaultCommander(log)
	osMgr := osmanager.NewDefaultOsManager(log, cmdr, fs)

	cfgPath, err := config.DefaultPath(osMgr, osMgr)
	require.NoError(t, err)

	cfg, err := config.Load(fs, cfgPath)
	require.NoError(t, err)

	return cfg
}

// setupIsolatedEnv sets XDG_CONFIG_HOME and XDG_CACHE_HOME to temp dirs for test isolation.
// It pre-creates the config directory so the lock file can be created by the locker.
func setupIsolatedEnv(t *testing.T) {
	t.Helper()

	configHome := filepath.Join(t.TempDir(), "config")
	cacheHome := filepath.Join(t.TempDir(), "cache")

	require.NoError(t, os.MkdirAll(filepath.Join(configHome, "cova"), 0o755))

	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_CACHE_HOME", cacheHome)
}

func TestAdd_AddingSingleCovenRepoShouldCreateCorrectSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")
	deps := newDeps(t)

	err := add.Run(t.Context(), deps, fileURL(repoDir), nil, "", false)
	require.NoError(t, err)

	cfg := loadConfig(t)
	require.Len(t, cfg.Subscriptions, 1)
	require.Equal(t, "acme-blocks", cfg.Subscriptions[0].Name)
	require.Equal(t, fileURL(repoDir), cfg.Subscriptions[0].Repo)
	require.Empty(t, cfg.Subscriptions[0].Path)
	require.Empty(t, cfg.Subscriptions[0].Ref)
}

func TestAdd_AddingSingleCovenRepoWithExtraCovenArgsShouldIgnoreThem(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")
	deps := newDeps(t)

	err := add.Run(t.Context(), deps, fileURL(repoDir), []string{"extra1", "extra2"}, "", false)
	require.NoError(t, err)

	cfg := loadConfig(t)
	require.Len(t, cfg.Subscriptions, 1)
	require.Equal(t, "acme-blocks", cfg.Subscriptions[0].Name)
}

func TestAdd_AddingMultiCovenRepoWithCovenArgsShouldCreateSubscriptionsWithPaths(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	repoDir := createMultiCovenRepo(t, "acme", []string{"platform", "frontend", "backend"})
	deps := newDeps(t)

	err := add.Run(t.Context(), deps, fileURL(repoDir), []string{"platform", "frontend"}, "", false)
	require.NoError(t, err)

	cfg := loadConfig(t)
	require.Len(t, cfg.Subscriptions, 2)

	require.Equal(t, "acme-platform", cfg.Subscriptions[0].Name)
	require.Equal(t, "covens/platform", cfg.Subscriptions[0].Path)

	require.Equal(t, "acme-frontend", cfg.Subscriptions[1].Name)
	require.Equal(t, "covens/frontend", cfg.Subscriptions[1].Path)
}

func TestAdd_AddingMultiCovenRepoWithoutCovenArgsShouldReturnError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	repoDir := createMultiCovenRepo(t, "acme", []string{"platform", "frontend"})
	deps := newDeps(t)

	err := add.Run(t.Context(), deps, fileURL(repoDir), nil, "", false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple covens")
	require.Contains(t, err.Error(), "platform")
	require.Contains(t, err.Error(), "frontend")
}

func TestAdd_AddingSameRepoTwiceShouldBeNoOp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")
	deps := newDeps(t)

	require.NoError(t, add.Run(t.Context(), deps, fileURL(repoDir), nil, "", false))
	require.NoError(t, add.Run(t.Context(), deps, fileURL(repoDir), nil, "", false))

	cfg := loadConfig(t)
	require.Len(t, cfg.Subscriptions, 1)
	require.Equal(t, "acme-blocks", cfg.Subscriptions[0].Name)
}

func TestAdd_AddingSameRepoWithDifferentRefShouldUpdateSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")
	gitCmd(t, repoDir, "tag", "v1.0.0")

	deps := newDeps(t)

	require.NoError(t, add.Run(t.Context(), deps, fileURL(repoDir), nil, "", false))
	require.NoError(t, add.Run(t.Context(), deps, fileURL(repoDir), nil, "v1.0.0", false))

	cfg := loadConfig(t)
	require.Len(t, cfg.Subscriptions, 1)
	require.Equal(t, "acme-blocks", cfg.Subscriptions[0].Name)
	require.Equal(t, "v1.0.0", cfg.Subscriptions[0].Ref)
}

func TestAdd_AddingWithRefFlagShouldRecordRefInSubscription(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	repoDir := createSingleCovenRepo(t, "acme", "blocks")

	// Create a tag so the ref actually resolves during checkout.
	gitCmd(t, repoDir, "tag", "v2.0.0")

	deps := newDeps(t)

	err := add.Run(t.Context(), deps, fileURL(repoDir), nil, "v2.0.0", false)
	require.NoError(t, err)

	cfg := loadConfig(t)
	require.Len(t, cfg.Subscriptions, 1)
	require.Equal(t, "v2.0.0", cfg.Subscriptions[0].Ref)
}

func TestAdd_AddingInvalidRepoURLShouldReturnError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping E2E tests in short mode")
	}

	setupIsolatedEnv(t)

	deps := newDeps(t)

	err := add.Run(t.Context(), deps, "not-a-valid-url", nil, "", false)
	require.Error(t, err)
}
