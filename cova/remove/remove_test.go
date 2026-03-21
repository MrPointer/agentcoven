package remove

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

// configYAML is a minimal valid config for use in tests with one subscription.
const configYAML = `
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/platform.git
`

// configTwoSubsSameRepoYAML has two subscriptions using the same repo (multi-coven scenario).
const configTwoSubsSameRepoYAML = `
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/blocks.git
    path: covens/platform
  - name: acme-tools
    repo: https://github.com/acme/blocks.git
    path: covens/tools
`

// configTwoSubsDiffRepoYAML has two subscriptions using different repos.
const configTwoSubsDiffRepoYAML = `
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/platform.git
  - name: acme-tools
    repo: https://github.com/acme/tools.git
`

// manifestYAML is a minimal single-coven manifest for use in tests.
const manifestYAML = `
org: acme
covens: platform
`

// hasSuffix reports whether s ends with suffix.
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// makeDeps returns a Deps with mocks. The locker executes the function inline.
func makeDeps(
	fs utils.FileSystem,
	store state.BlockStore,
	dispatcher exporter.Dispatcher,
	envMgr osmanager.EnvironmentManager,
	userMgr osmanager.UserManager,
) Deps {
	locker := &utils.MoqLocker{
		WithLockFunc: func(ctx context.Context, _ string, fn func() error) error {
			return fn()
		},
	}

	return Deps{
		Logger:      logger.NoopLogger{},
		FileSystem:  fs,
		Locker:      locker,
		BlockStore:  store,
		Dispatcher:  dispatcher,
		EnvManager:  envMgr,
		UserManager: userMgr,
	}
}

// makeEnvUserMgr returns standard env/user manager mocks returning no XDG overrides
// and home dir /home/user.
func makeEnvUserMgr() (osmanager.EnvironmentManager, osmanager.UserManager) {
	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	return envMgr, userMgr
}

// makeNoOpFS returns a filesystem mock that serves the given configYAML for config reads,
// manifestYAML for manifest reads, returns true for PathExists, and no-ops writes.
func makeNoOpFS(cfgYAML string) *utils.MoqFileSystem {
	return &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			switch {
			case hasSuffix(path, "config.yaml"):
				return []byte(cfgYAML), nil
			case hasSuffix(path, "manifest.yaml"):
				return []byte(manifestYAML), nil
			default:
				return nil, errors.New("unexpected path: " + path)
			}
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
		RemovePathFunc:          func(path string) error { return nil },
	}
}

// makeNoRecordsStore returns a BlockStore that returns no records for any subscription.
func makeNoRecordsStore() *state.MoqBlockStore {
	return &state.MoqBlockStore{
		QueryBySubscriptionFunc: func(ctx context.Context, subscription string) ([]state.Record, error) {
			return nil, nil
		},
		DeleteBySubscriptionFunc: func(ctx context.Context, subscription string) error {
			return nil
		},
	}
}

// makeOneRecordStore returns a BlockStore with one record for "acme-platform".
func makeOneRecordStore(records []state.Record) *state.MoqBlockStore {
	return &state.MoqBlockStore{
		QueryBySubscriptionFunc: func(ctx context.Context, subscription string) ([]state.Record, error) {
			return records, nil
		},
		DeleteBySubscriptionFunc: func(ctx context.Context, subscription string) error {
			return nil
		},
	}
}

// makeNoOpDispatcher returns a Dispatcher that succeeds on Remove.
func makeNoOpDispatcher() *exporter.MoqDispatcher {
	return &exporter.MoqDispatcher{
		RemoveFunc: func(ctx context.Context, agent string, req *exporter.RemoveRequest) (*exporter.RemoveResponse, error) {
			return &exporter.RemoveResponse{}, nil
		},
	}
}

func TestRun_RemovingSubscriptionShouldErrorWhenNoneOfTheNamesExistInConfig(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeNoOpFS(configYAML)
	store := makeNoRecordsStore()
	deps := makeDeps(fs, store, makeNoOpDispatcher(), envMgr, userMgr)

	err := Run(ctx, deps, []string{"does-not-exist"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "none of the provided subscription names")
}

func TestRun_RemovingSubscriptionShouldWarnAndSkipWhenNameNotFoundInConfig(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeNoOpFS(configYAML)
	store := makeNoRecordsStore()

	var warnings []string

	log := &logger.MoqLogger{
		WarningFunc: func(format string, args ...any) {
			warnings = append(warnings, format)
		},
		SuccessFunc: func(format string, args ...any) {},
	}

	locker := &utils.MoqLocker{
		WithLockFunc: func(ctx context.Context, _ string, fn func() error) error { return fn() },
	}
	deps := Deps{
		Logger:      log,
		FileSystem:  fs,
		Locker:      locker,
		BlockStore:  store,
		Dispatcher:  makeNoOpDispatcher(),
		EnvManager:  envMgr,
		UserManager: userMgr,
	}

	// One valid name, one missing name — should succeed because one was found.
	err := Run(ctx, deps, []string{"acme-platform", "no-such-sub"})

	require.NoError(t, err)
	require.NotEmpty(t, warnings, "expected a warning for the missing subscription")
}

func TestRun_RemovingSubscriptionShouldSucceedWithNoRecordsAndNoWorkspaceCleanup(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	var removedPaths []string

	// First load returns acme-platform; subsequent loads (after removal) return empty config.
	configLoadCount := 0
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if hasSuffix(path, "config.yaml") {
				configLoadCount++
				if configLoadCount == 1 {
					return []byte(configYAML), nil
				}

				return []byte("subscriptions: []\n"), nil
			}

			return []byte(manifestYAML), nil
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
		RemovePathFunc: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	store := makeNoRecordsStore()
	deps := makeDeps(fs, store, makeNoOpDispatcher(), envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	// No records means no placed files deleted; workspace removed because no remaining subs reference same repo.
	require.Empty(t, store.DeleteBySubscriptionCalls(), "no state deletion without records")
	require.Contains(
		t,
		removedPaths,
		"/home/user/.cache/cova/repos/github.com/acme/platform",
		"workspace should be deleted",
	)
}

func TestRun_RemovingSubscriptionShouldDeletePlacedFilesAndStateRecords(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	records := []state.Record{
		{
			Path:         "/home/user/.claude/skills/my-skill.md",
			Subscription: "acme-platform",
			Source:       "skills/my-skill",
			BlockType:    "skills",
			Agent:        "claude-code",
		},
	}

	var removedPaths []string

	fs := makeNoOpFS(configYAML)
	fs.RemovePathFunc = func(path string) error {
		removedPaths = append(removedPaths, path)
		return nil
	}

	store := makeOneRecordStore(records)
	dispatcher := makeNoOpDispatcher()
	deps := makeDeps(fs, store, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	require.Contains(t, removedPaths, "/home/user/.claude/skills/my-skill.md", "placed file should be deleted")
	require.Len(t, store.DeleteBySubscriptionCalls(), 1, "state records should be deleted")
	require.Equal(t, "acme-platform", store.DeleteBySubscriptionCalls()[0].Subscription)
}

func TestRun_RemovingSubscriptionShouldNotifyExporterForEachAgent(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	records := []state.Record{
		{
			Path:         "/home/user/.claude/skills/my-skill.md",
			Subscription: "acme-platform",
			Source:       "skills/my-skill",
			BlockType:    "skills",
			Agent:        "claude-code",
		},
		{
			Path:         "/home/user/.cursor/skills/my-skill.md",
			Subscription: "acme-platform",
			Source:       "skills/my-skill",
			BlockType:    "skills",
			Agent:        "cursor",
		},
	}

	fs := makeNoOpFS(configYAML)

	store := makeOneRecordStore(records)

	var dispatchedAgents []string

	dispatcher := &exporter.MoqDispatcher{
		RemoveFunc: func(ctx context.Context, agent string, req *exporter.RemoveRequest) (*exporter.RemoveResponse, error) {
			dispatchedAgents = append(dispatchedAgents, agent)
			return &exporter.RemoveResponse{}, nil
		},
	}
	deps := makeDeps(fs, store, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	require.Len(t, dispatchedAgents, 2, "one Remove call per agent")
	require.ElementsMatch(t, []string{"claude-code", "cursor"}, dispatchedAgents)
}

func TestRun_RemovingSubscriptionShouldWarnOnExporterErrorAndContinue(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	records := []state.Record{
		{
			Path:         "/home/user/.claude/skills/my-skill.md",
			Subscription: "acme-platform",
			Source:       "skills/my-skill",
			BlockType:    "skills",
			Agent:        "claude-code",
		},
	}

	fs := makeNoOpFS(configYAML)

	store := makeOneRecordStore(records)

	dispatcher := &exporter.MoqDispatcher{
		RemoveFunc: func(ctx context.Context, agent string, req *exporter.RemoveRequest) (*exporter.RemoveResponse, error) {
			return nil, errors.New("exporter failed")
		},
	}

	var warnings []string

	log := &logger.MoqLogger{
		WarningFunc: func(format string, args ...any) { warnings = append(warnings, format) },
		SuccessFunc: func(format string, args ...any) {},
	}
	locker := &utils.MoqLocker{
		WithLockFunc: func(ctx context.Context, _ string, fn func() error) error { return fn() },
	}
	deps := Deps{
		Logger:      log,
		FileSystem:  fs,
		Locker:      locker,
		BlockStore:  store,
		Dispatcher:  dispatcher,
		EnvManager:  envMgr,
		UserManager: userMgr,
	}

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	require.NotEmpty(t, warnings, "expected warning about exporter failure")
	// State records should still be deleted despite exporter error.
	require.Len(t, store.DeleteBySubscriptionCalls(), 1)
}

func TestRun_RemovingSubscriptionShouldSkipExporterNotificationWhenWorkspaceIsMissing(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	records := []state.Record{
		{
			Path:         "/home/user/.claude/skills/my-skill.md",
			Subscription: "acme-platform",
			Source:       "skills/my-skill",
			BlockType:    "skills",
			Agent:        "claude-code",
		},
	}

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			// Config exists; workspace does not.
			if hasSuffix(path, "config.yaml") {
				return true, nil
			}

			return false, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if hasSuffix(path, "config.yaml") {
				return []byte(configYAML), nil
			}

			return nil, errors.New("unexpected path: " + path)
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
		RemovePathFunc:          func(path string) error { return nil },
	}

	store := makeOneRecordStore(records)

	dispatcherCalled := false
	dispatcher := &exporter.MoqDispatcher{
		RemoveFunc: func(ctx context.Context, agent string, req *exporter.RemoveRequest) (*exporter.RemoveResponse, error) {
			dispatcherCalled = true
			return &exporter.RemoveResponse{}, nil
		},
	}
	deps := makeDeps(fs, store, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	require.False(t, dispatcherCalled, "dispatcher should not be called when workspace is missing")
	// Files and state should still be cleaned up.
	require.Len(t, store.DeleteBySubscriptionCalls(), 1)
}

func TestRun_RemovingSubscriptionShouldWarnOnFileDeletionFailureAndContinue(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	records := []state.Record{
		{
			Path:         "/home/user/.claude/skills/my-skill.md",
			Subscription: "acme-platform",
			Source:       "skills/my-skill",
			BlockType:    "skills",
			Agent:        "claude-code",
		},
	}

	var warnings []string

	fs := makeNoOpFS(configYAML)
	fs.RemovePathFunc = func(path string) error {
		if hasSuffix(path, "my-skill.md") {
			return errors.New("permission denied")
		}

		return nil
	}

	store := makeOneRecordStore(records)

	log := &logger.MoqLogger{
		WarningFunc: func(format string, args ...any) { warnings = append(warnings, format) },
		SuccessFunc: func(format string, args ...any) {},
	}
	locker := &utils.MoqLocker{
		WithLockFunc: func(ctx context.Context, _ string, fn func() error) error { return fn() },
	}
	deps := Deps{
		Logger:      log,
		FileSystem:  fs,
		Locker:      locker,
		BlockStore:  store,
		Dispatcher:  makeNoOpDispatcher(),
		EnvManager:  envMgr,
		UserManager: userMgr,
	}

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	require.NotEmpty(t, warnings, "expected warning about file deletion failure")
	// State records should still be deleted.
	require.Len(t, store.DeleteBySubscriptionCalls(), 1)
}

func TestRun_RemovingSubscriptionShouldKeepWorkspaceWhenOtherSubsReferSameRepo(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	// Config starts with two subs referencing same repo. After removing acme-platform,
	// acme-tools still references the same repo so workspace must be kept.
	callCount := 0
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if hasSuffix(path, "config.yaml") {
				if callCount == 0 {
					// First load: both subscriptions present.
					callCount++
					return []byte(configTwoSubsSameRepoYAML), nil
				}
				// Subsequent loads: acme-platform removed, acme-tools remains.
				return []byte(`
subscriptions:
  - name: acme-tools
    repo: https://github.com/acme/blocks.git
    path: covens/tools
`), nil
			}

			return []byte(manifestYAML), nil
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
		RemovePathFunc:          func(path string) error { return nil },
	}

	var removedPaths []string

	fs.RemovePathFunc = func(path string) error {
		removedPaths = append(removedPaths, path)
		return nil
	}

	store := makeNoRecordsStore()
	deps := makeDeps(fs, store, makeNoOpDispatcher(), envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	// Workspace should NOT be deleted — acme-tools still references the same repo.
	for _, p := range removedPaths {
		require.NotContains(t, p, "github.com/acme/blocks", "workspace should be kept when shared")
	}
}

func TestRun_RemovingSubscriptionShouldDeleteWorkspaceWhenNoOtherSubsReferSameRepo(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	// Config has two subs with different repos. After removing acme-platform,
	// only acme-tools remains and it references a different repo.
	callCount := 0
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if hasSuffix(path, "config.yaml") {
				if callCount == 0 {
					callCount++
					return []byte(configTwoSubsDiffRepoYAML), nil
				}

				return []byte(`
subscriptions:
  - name: acme-tools
    repo: https://github.com/acme/tools.git
`), nil
			}

			return []byte(manifestYAML), nil
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
	}

	var removedPaths []string

	fs.RemovePathFunc = func(path string) error {
		removedPaths = append(removedPaths, path)
		return nil
	}

	store := makeNoRecordsStore()
	deps := makeDeps(fs, store, makeNoOpDispatcher(), envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	// Workspace for acme-platform should be deleted.
	require.Contains(t, removedPaths, "/home/user/.cache/cova/repos/github.com/acme/platform")
}

func TestRun_RemovingMultipleSubscriptionsShouldProcessEachIndependently(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	const twoSubsConfig = `
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/platform.git
  - name: acme-tools
    repo: https://github.com/acme/tools.git
`

	// Track how many config loads happen (for reload-after-remove).
	configLoadCount := 0
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if hasSuffix(path, "config.yaml") {
				configLoadCount++

				switch configLoadCount {
				case 1:
					return []byte(twoSubsConfig), nil
				case 2:
					// After removing acme-platform, reload sees only acme-tools.
					return []byte(`
subscriptions:
  - name: acme-tools
    repo: https://github.com/acme/tools.git
`), nil
				default:
					// After removing acme-tools, reload sees empty.
					return []byte(`subscriptions: []`), nil
				}
			}

			return []byte(manifestYAML), nil
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
		RemovePathFunc:          func(path string) error { return nil },
	}

	var deleteSubCalls []string

	store := &state.MoqBlockStore{
		QueryBySubscriptionFunc: func(ctx context.Context, subscription string) ([]state.Record, error) {
			return nil, nil
		},
		DeleteBySubscriptionFunc: func(ctx context.Context, subscription string) error {
			deleteSubCalls = append(deleteSubCalls, subscription)
			return nil
		},
	}

	deps := makeDeps(fs, store, makeNoOpDispatcher(), envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform", "acme-tools"})

	require.NoError(t, err)
	// No state records, so DeleteBySubscription should not be called.
	require.Empty(t, deleteSubCalls)
}

func TestRun_RemovingSubscriptionShouldErrorWhenConfigLoadFails(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return nil, errors.New("disk read error")
		},
	}

	deps := makeDeps(fs, nil, nil, envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "loading config")
}

func TestRun_RemovingSubscriptionShouldGroupBlocksByBlockTypeInRemoveRequest(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	records := []state.Record{
		{
			Path:         "/home/user/.claude/skills/my-skill.md",
			Subscription: "acme-platform",
			Source:       "skills/my-skill",
			BlockType:    "skills",
			Agent:        "claude-code",
		},
		{
			Path:         "/home/user/.claude/agents/my-agent.md",
			Subscription: "acme-platform",
			Source:       "agents/my-agent",
			BlockType:    "agents",
			Agent:        "claude-code",
		},
	}

	fs := makeNoOpFS(configYAML)

	store := makeOneRecordStore(records)

	var capturedRequests []*exporter.RemoveRequest

	dispatcher := &exporter.MoqDispatcher{
		RemoveFunc: func(ctx context.Context, agent string, req *exporter.RemoveRequest) (*exporter.RemoveResponse, error) {
			capturedRequests = append(capturedRequests, req)
			return &exporter.RemoveResponse{}, nil
		},
	}
	deps := makeDeps(fs, store, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	require.Len(t, capturedRequests, 1, "one Remove call for the single agent")

	req := capturedRequests[0]
	require.Contains(t, req.Blocks, "skills", "remove request must contain skills block type")
	require.Contains(t, req.Blocks, "agents", "remove request must contain agents block type")
	require.Equal(t, "acme", req.Manifest.Org)
	require.Equal(t, "platform", req.Manifest.Coven)
	require.Equal(t, "remove", req.Operation)
	require.Equal(t, "acme-platform", req.Subscription)
}
