package add

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/adapter"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/manifest"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

func TestBuildSubscriptions_BuildingSingleCovenShouldReturnOneSubscription(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform"}, true)

	subs, err := BuildSubscriptions(mf, "https://github.com/acme/blocks.git", "", nil)

	require.NoError(t, err)
	require.Len(t, subs, 1)
	require.Equal(t, "acme-platform", subs[0].Name)
	require.Equal(t, "https://github.com/acme/blocks.git", subs[0].Repo)
	require.Empty(t, subs[0].Path)
	require.Empty(t, subs[0].Ref)
}

func TestBuildSubscriptions_BuildingSingleCovenWithRefShouldSetRef(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform"}, true)

	subs, err := BuildSubscriptions(mf, "https://github.com/acme/blocks.git", "v1.0.0", nil)

	require.NoError(t, err)
	require.Len(t, subs, 1)
	require.Equal(t, "v1.0.0", subs[0].Ref)
}

func TestBuildSubscriptions_BuildingSingleCovenShouldIgnoreExtraCovenArgs(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform"}, true)

	subs, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "", []string{"extra", "args"},
	)

	require.NoError(t, err)
	require.Len(t, subs, 1)
	require.Equal(t, "acme-platform", subs[0].Name)
}

func TestBuildSubscriptions_BuildingMultiCovenWithValidArgsShouldReturnSubscriptions(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend", "backend"}, false)

	subs, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "", []string{"platform", "frontend"},
	)

	require.NoError(t, err)
	require.Len(t, subs, 2)
	require.Equal(t, "acme-platform", subs[0].Name)
	require.Equal(t, "covens/platform", subs[0].Path)
	require.Equal(t, "acme-frontend", subs[1].Name)
	require.Equal(t, "covens/frontend", subs[1].Path)
}

func TestBuildSubscriptions_BuildingMultiCovenWithNoArgsShouldReturnError(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend"}, false)

	_, err := BuildSubscriptions(mf, "https://github.com/acme/blocks.git", "", nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "multiple covens")
	require.Contains(t, err.Error(), "platform")
	require.Contains(t, err.Error(), "frontend")
}

func TestBuildSubscriptions_BuildingMultiCovenWithUnknownNameShouldReturnError(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend"}, false)

	_, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "", []string{"nonexistent"},
	)

	require.Error(t, err)
	require.Contains(t, err.Error(), "nonexistent")
	require.Contains(t, err.Error(), "not found")
}

func TestBuildSubscriptions_BuildingMultiCovenWithRefShouldSetRefOnAll(t *testing.T) {
	mf := manifest.NewRootManifest("acme", []string{"platform", "frontend"}, false)

	subs, err := BuildSubscriptions(
		mf, "https://github.com/acme/blocks.git", "main", []string{"platform", "frontend"},
	)

	require.NoError(t, err)
	require.Len(t, subs, 2)
	require.Equal(t, "main", subs[0].Ref)
	require.Equal(t, "main", subs[1].Ref)
}

// singleCovenManifestYAML is a minimal single-coven manifest for add tests.
const singleCovenManifestYAML = `
org: acme
covens: platform
`

// makeAddDeps returns Deps wired with the provided mocks.
// The Locker executes the function inline (no actual locking).
func makeAddDeps(
	fs utils.FileSystem,
	git workspace.Git,
	store state.BlockStore,
	dispatcher adapter.Dispatcher,
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
		Git:         git,
		BlockStore:  store,
		Dispatcher:  dispatcher,
		EnvManager:  envMgr,
		UserManager: userMgr,
	}
}

// basicMocks bundles the mocks sufficient for a single-coven add that returns UpsertAdded.
// The filesystem serves a manifest at the workspace path.
// The config file does not exist (empty config — all subscriptions will be added).
type basicMocks struct {
	fs      utils.FileSystem
	git     workspace.Git
	envMgr  osmanager.EnvironmentManager
	userMgr osmanager.UserManager
}

// makeBasicMocks returns mocks sufficient for a single-coven add that returns UpsertAdded.
func makeBasicMocks() basicMocks {
	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	git := &workspace.MoqGit{
		FetchFunc: func(ctx context.Context, repoDir string) error { return nil },
	}

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			// Workspace exists; config file does not.
			if strings.HasSuffix(path, "config.yaml") {
				return false, nil
			}

			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte(singleCovenManifestYAML), nil
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
	}

	return basicMocks{fs: fs, git: git, envMgr: envMgr, userMgr: userMgr}
}

func TestRun_RunningAddWithApplyFalseShouldNotCallDispatcher(t *testing.T) {
	ctx := t.Context()
	m := makeBasicMocks()

	dispatcherCalled := false
	dispatcher := &adapter.MoqDispatcher{
		ApplyFunc: func(_ context.Context, _ string, _ *adapter.ApplyRequest) (*adapter.ApplyResponse, error) {
			dispatcherCalled = true
			return &adapter.ApplyResponse{}, nil
		},
	}

	deps := makeAddDeps(m.fs, m.git, nil, dispatcher, m.envMgr, m.userMgr)

	err := Run(ctx, deps, "https://github.com/acme/platform.git", nil, "", false)

	require.NoError(t, err)
	require.False(t, dispatcherCalled, "dispatcher should not be called when apply is false")
}

func TestRun_RunningAddWithApplyTrueAndNoOpSubscriptionShouldNotCallDispatcher(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}
	git := &workspace.MoqGit{
		FetchFunc: func(ctx context.Context, repoDir string) error { return nil },
	}

	// Config already contains the subscription — upsert will be a no-op.
	existingConfigYAML := `
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/platform.git
    ref: ""
`
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, "manifest.yaml") {
				return []byte(singleCovenManifestYAML), nil
			}

			return []byte(existingConfigYAML), nil
		},
		CreateDirectoryFunc: func(path string) error { return nil },
	}

	dispatcherCalled := false
	dispatcher := &adapter.MoqDispatcher{
		ApplyFunc: func(_ context.Context, _ string, _ *adapter.ApplyRequest) (*adapter.ApplyResponse, error) {
			dispatcherCalled = true
			return &adapter.ApplyResponse{}, nil
		},
	}

	deps := makeAddDeps(fs, git, nil, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, "https://github.com/acme/platform.git", nil, "", true)

	require.NoError(t, err)
	require.False(t, dispatcherCalled, "dispatcher should not be called when all subscriptions are no-ops")
}

func TestRun_RunningAddWithApplyTrueAndNewSubscriptionShouldReturnDistinguishableErrorOnApplyFailure(t *testing.T) {
	ctx := t.Context()
	m := makeBasicMocks()

	// Config with frameworks so apply can proceed past the frameworks check, but
	// the dispatcher will fail — this exercises the error wrapping path.
	configWithFrameworks := `
frameworks:
  - claude-code
`
	fsWithConfig := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			// Workspace exists; config file exists.
			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if strings.HasSuffix(path, "manifest.yaml") {
				return []byte(singleCovenManifestYAML), nil
			}

			if strings.HasSuffix(path, "config.yaml") {
				// First call (for upsert): empty config so subscription is added.
				// Second call (for apply): config with frameworks.
				return []byte(configWithFrameworks), nil
			}

			return nil, errors.New("unexpected path: " + path)
		},
		CreateDirectoryFunc:     func(path string) error { return nil },
		CreateTemporaryFileFunc: func(dir, pattern string) (string, error) { return "/tmp/cfg.tmp", nil },
		WriteFileFunc:           func(path string, r io.Reader) (int64, error) { return 0, nil },
		RenameFunc:              func(old, dst string) error { return nil },
	}

	// Override the PathExists so the upsert loads an empty config file.
	callCount := 0

	fsWithConfig.PathExistsFunc = func(path string) (bool, error) {
		if strings.HasSuffix(path, "config.yaml") {
			callCount++
			if callCount == 1 {
				// First check during upsert: file does not exist → empty config.
				return false, nil
			}
		}

		return true, nil
	}

	fsWithConfig.ReadFileContentsFunc = func(path string) ([]byte, error) {
		if strings.HasSuffix(path, "manifest.yaml") {
			return []byte(singleCovenManifestYAML), nil
		}

		// config.yaml reads after the upsert (during apply)
		return []byte(configWithFrameworks), nil
	}

	applyErr := errors.New("adapter exploded")
	dispatcher := &adapter.MoqDispatcher{
		ApplyFunc: func(_ context.Context, _ string, _ *adapter.ApplyRequest) (*adapter.ApplyResponse, error) {
			return nil, applyErr
		},
	}

	deps := makeAddDeps(fsWithConfig, m.git, nil, dispatcher, m.envMgr, m.userMgr)

	err := Run(ctx, deps, "https://github.com/acme/platform.git", nil, "", true)

	require.Error(t, err)
	require.Contains(t, err.Error(), "subscriptions added successfully")
	require.Contains(t, err.Error(), "apply failed")
}

func TestLogUpsertResult_LoggingShouldUseCorrectLevel(t *testing.T) {
	tests := []struct {
		name           string
		expectedSubstr string
		result         config.UpsertResult
		expectSuccess  bool
		expectInfo     bool
	}{
		{"WhenAdded", "added", config.UpsertAdded, true, false},
		{"WhenUpdated", "updated", config.UpsertUpdated, false, true},
		{"WhenNoOp", "already up to date", config.UpsertNoOp, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var successMsg, infoMsg string

			mock := &logger.MoqLogger{
				SuccessFunc: func(format string, _ ...any) {
					successMsg = format
				},
				InfoFunc: func(format string, _ ...any) {
					infoMsg = format
				},
				TraceFunc:   func(string, ...any) {},
				DebugFunc:   func(string, ...any) {},
				WarningFunc: func(string, ...any) {},
				ErrorFunc:   func(string, ...any) {},
				CloseFunc:   func() error { return nil },
			}

			LogUpsertResult(mock, "test-sub", tt.result)

			if tt.expectSuccess {
				require.Contains(t, successMsg, tt.expectedSubstr)
			}

			if tt.expectInfo {
				require.Contains(t, infoMsg, tt.expectedSubstr)
			}
		})
	}
}
