package apply

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

// configYAML is a minimal valid config for use in tests.
const configYAML = `
agents:
  - claude-code
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/platform.git
    ref: ""
`

// configNoAgentsYAML is a config with no agents.
const configNoAgentsYAML = `
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/platform.git
`

// manifestYAML is a minimal single-coven manifest.
const manifestYAML = `
org: acme
covens: platform
`

// readFileByName returns the appropriate YAML content based on the file name in path.
// It returns configYAML for config.yaml reads and manifestYAML for manifest.yaml reads.
func readFileByName(path string) ([]byte, error) {
	switch {
	case hasSuffix(path, "config.yaml"):
		return []byte(configYAML), nil
	case hasSuffix(path, "manifest.yaml"):
		return []byte(manifestYAML), nil
	default:
		return nil, errors.New("unexpected path: " + path)
	}
}

// hasSuffix reports whether s ends with suffix.
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// pathExistsNoVariants returns true for all paths except variants.yaml files,
// allowing block discovery tests to proceed without variant resolution side-effects.
func pathExistsNoVariants(path string) (bool, error) {
	if hasSuffix(path, "variants.yaml") {
		return false, nil
	}

	return true, nil
}

// makeDeps returns a Deps with all mocks pre-filled with sensible defaults.
// Override individual fields as needed per test.
func makeDeps(
	fs utils.FileSystem,
	store state.BlockStore,
	git workspace.Git,
	dispatcher exporter.Dispatcher,
	envMgr osmanager.EnvironmentManager,
	userMgr osmanager.UserManager,
) Deps {
	return Deps{
		Logger:      logger.NoopLogger{},
		FileSystem:  fs,
		Locker:      nil,
		Git:         git,
		BlockStore:  store,
		Dispatcher:  dispatcher,
		EnvManager:  envMgr,
		UserManager: userMgr,
	}
}

func TestRun_RunningApplyShouldReturnErrorWhenConfigHasNoAgents(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte(configNoAgentsYAML), nil
		},
	}
	deps := makeDeps(fs, nil, nil, nil, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "no agents configured")
}

func TestRun_RunningApplyShouldReturnErrorWhenSubscriptionNameNotFound(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte(configYAML), nil
		},
	}
	deps := makeDeps(fs, nil, nil, nil, envMgr, userMgr)

	err := Run(ctx, deps, []string{"does-not-exist"})

	require.Error(t, err)
	require.Contains(t, err.Error(), "not found in config")
}

func TestRun_RunningApplyShouldSkipSubscriptionWhenWorkspaceIsMissing(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	pathExistsCallCount := 0
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			pathExistsCallCount++
			if pathExistsCallCount == 1 {
				// config file exists
				return true, nil
			}
			// workspace does not exist
			return false, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte(configYAML), nil
		},
	}
	deps := makeDeps(fs, nil, nil, nil, envMgr, userMgr)

	// Should not return error — subscription is skipped with warning
	err := Run(ctx, deps, nil)

	require.NoError(t, err)
}

func TestRun_RunningApplyShouldApplyBlocksWhenEverythingIsPresent(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	// Track what files were "created" or "copied".
	var (
		createdDirs []string
		copiedFiles [][2]string
	)

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			// Variants files do not exist (agnostic block).
			if hasSuffix(path, "variants.yaml") {
				return false, nil
			}
			// Everything exists except the target placement path.
			if path == "/output/.claude/skills/my-skill.md" {
				return false, nil
			}

			return true, nil
		},
		ReadFileContentsFunc: readFileByName,
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			// Return one block type directory and one block.
			switch path {
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree":
				return []os.DirEntry{&fakeDirEntry{name: "skills", isDir: true}}, nil
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree/skills":
				return []os.DirEntry{&fakeDirEntry{name: "my-skill", isDir: true}}, nil
			default:
				return nil, nil
			}
		},
		CreateDirectoryFunc: func(path string) error {
			createdDirs = append(createdDirs, path)
			return nil
		},
		CopyFileFunc: func(src, dst string) (int64, error) {
			copiedFiles = append(copiedFiles, [2]string{src, dst})
			return 100, nil
		},
	}

	git := &workspace.MoqGit{
		WorktreeAddFunc: func(ctx context.Context, repoDir, ref string) (string, error) {
			return repoDir + "/tempdir-worktree", nil
		},
	}

	store := &state.MoqBlockStore{
		QueryByPathFunc: func(ctx context.Context, path string) (*state.Record, error) {
			return nil, state.ErrNotFound
		},
		QueryBySubscriptionAgentFunc: func(ctx context.Context, subscription, agent string) ([]state.Record, error) {
			return nil, nil
		},
		RecordBatchFunc: func(ctx context.Context, records []state.Record) error {
			return nil
		},
	}

	exporterResp := &exporter.ApplyResponse{
		Results: []exporter.BlockResult{
			{
				Name: "my-skill",
				Placements: []exporter.Placement{
					{
						Path:   "/output/.claude/skills/my-skill.md",
						Source: "skills/my-skill/my-skill.md",
					},
				},
			},
		},
	}
	dispatcher := &exporter.MoqDispatcher{
		ApplyFunc: func(ctx context.Context, agent string, req *exporter.ApplyRequest) (*exporter.ApplyResponse, error) {
			return exporterResp, nil
		},
	}

	deps := makeDeps(fs, store, git, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.NoError(t, err)
	require.Len(t, copiedFiles, 1)
	require.Equal(t, "/output/.claude/skills/my-skill.md", copiedFiles[0][1])
	require.Len(t, store.RecordBatchCalls(), 1)
	require.Len(t, store.RecordBatchCalls()[0].Records, 1)
	require.Equal(t, "/output/.claude/skills/my-skill.md", store.RecordBatchCalls()[0].Records[0].Path)
	require.Equal(t, "acme-platform", store.RecordBatchCalls()[0].Records[0].Subscription)
	require.Equal(t, "claude-code", store.RecordBatchCalls()[0].Records[0].Agent)
	require.Empty(t, store.RecordBatchCalls()[0].Records[0].Checksum)
}

func TestRun_RunningApplyShouldSkipBlockWhenExporterReportsError(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	fs := &utils.MoqFileSystem{
		PathExistsFunc:       func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: readFileByName,
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			switch path {
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree":
				return []os.DirEntry{&fakeDirEntry{name: "skills", isDir: true}}, nil
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree/skills":
				return []os.DirEntry{&fakeDirEntry{name: "my-skill", isDir: true}}, nil
			default:
				return nil, nil
			}
		},
	}

	git := &workspace.MoqGit{
		WorktreeAddFunc: func(ctx context.Context, repoDir, ref string) (string, error) {
			return repoDir + "/tempdir-worktree", nil
		},
	}

	store := &state.MoqBlockStore{
		QueryBySubscriptionAgentFunc: func(ctx context.Context, subscription, agent string) ([]state.Record, error) {
			return nil, nil
		},
		RecordBatchFunc: func(ctx context.Context, records []state.Record) error {
			return nil
		},
	}

	blockErr := "unsupported block type"
	exporterResp := &exporter.ApplyResponse{
		Results: []exporter.BlockResult{
			{
				Name:  "my-skill",
				Error: &blockErr,
			},
		},
	}
	dispatcher := &exporter.MoqDispatcher{
		ApplyFunc: func(ctx context.Context, agent string, req *exporter.ApplyRequest) (*exporter.ApplyResponse, error) {
			return exporterResp, nil
		},
	}

	deps := makeDeps(fs, store, git, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.NoError(t, err)
	// No files were copied, no state recorded.
	require.Empty(t, store.RecordBatchCalls())
}

func TestRun_RunningApplyShouldSkipPlacementOnUserFileConflict(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	var copiedFiles [][2]string

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			// Variants files do not exist (agnostic block).
			if hasSuffix(path, "variants.yaml") {
				return false, nil
			}
			// Workspace and worktree always exist; placement path exists too (user file).
			return true, nil
		},
		ReadFileContentsFunc: readFileByName,
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			switch path {
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree":
				return []os.DirEntry{&fakeDirEntry{name: "skills", isDir: true}}, nil
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree/skills":
				return []os.DirEntry{&fakeDirEntry{name: "my-skill", isDir: true}}, nil
			default:
				return nil, nil
			}
		},
		CopyFileFunc: func(src, dst string) (int64, error) {
			copiedFiles = append(copiedFiles, [2]string{src, dst})
			return 100, nil
		},
	}

	git := &workspace.MoqGit{
		WorktreeAddFunc: func(ctx context.Context, repoDir, ref string) (string, error) {
			return repoDir + "/tempdir-worktree", nil
		},
	}

	store := &state.MoqBlockStore{
		// Path exists but is not tracked — user file conflict.
		QueryByPathFunc: func(ctx context.Context, path string) (*state.Record, error) {
			return nil, state.ErrNotFound
		},
		QueryBySubscriptionAgentFunc: func(ctx context.Context, subscription, agent string) ([]state.Record, error) {
			return nil, nil
		},
		RecordBatchFunc: func(ctx context.Context, records []state.Record) error {
			return nil
		},
	}

	exporterResp := &exporter.ApplyResponse{
		Results: []exporter.BlockResult{
			{
				Name: "my-skill",
				Placements: []exporter.Placement{
					{
						Path:   "/output/.claude/skills/my-skill.md",
						Source: "skills/my-skill/my-skill.md",
					},
				},
			},
		},
	}
	dispatcher := &exporter.MoqDispatcher{
		ApplyFunc: func(ctx context.Context, agent string, req *exporter.ApplyRequest) (*exporter.ApplyResponse, error) {
			return exporterResp, nil
		},
	}

	deps := makeDeps(fs, store, git, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.NoError(t, err)
	require.Empty(t, copiedFiles, "no files should be copied when a user file conflict is detected")
	require.Empty(t, store.RecordBatchCalls())
}

func TestRun_RunningApplyShouldSkipPlacementOnCrossSubscriptionConflict(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	var copiedFiles [][2]string

	fs := &utils.MoqFileSystem{
		PathExistsFunc:       pathExistsNoVariants,
		ReadFileContentsFunc: readFileByName,
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			switch path {
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree":
				return []os.DirEntry{&fakeDirEntry{name: "skills", isDir: true}}, nil
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree/skills":
				return []os.DirEntry{&fakeDirEntry{name: "my-skill", isDir: true}}, nil
			default:
				return nil, nil
			}
		},
		CopyFileFunc: func(src, dst string) (int64, error) {
			copiedFiles = append(copiedFiles, [2]string{src, dst})
			return 100, nil
		},
	}

	git := &workspace.MoqGit{
		WorktreeAddFunc: func(ctx context.Context, repoDir, ref string) (string, error) {
			return repoDir + "/tempdir-worktree", nil
		},
	}

	store := &state.MoqBlockStore{
		// Path is tracked by a different subscription.
		QueryByPathFunc: func(ctx context.Context, path string) (*state.Record, error) {
			return &state.Record{
				Path:         path,
				Subscription: "other-subscription",
				Agent:        "claude-code",
			}, nil
		},
		QueryBySubscriptionAgentFunc: func(ctx context.Context, subscription, agent string) ([]state.Record, error) {
			return nil, nil
		},
		RecordBatchFunc: func(ctx context.Context, records []state.Record) error {
			return nil
		},
	}

	exporterResp := &exporter.ApplyResponse{
		Results: []exporter.BlockResult{
			{
				Name: "my-skill",
				Placements: []exporter.Placement{
					{
						Path:   "/output/.claude/skills/my-skill.md",
						Source: "skills/my-skill/my-skill.md",
					},
				},
			},
		},
	}
	dispatcher := &exporter.MoqDispatcher{
		ApplyFunc: func(ctx context.Context, agent string, req *exporter.ApplyRequest) (*exporter.ApplyResponse, error) {
			return exporterResp, nil
		},
	}

	deps := makeDeps(fs, store, git, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.NoError(t, err)
	require.Empty(t, copiedFiles, "no files should be copied when a cross-subscription conflict is detected")
	require.Empty(t, store.RecordBatchCalls())
}

func TestRun_RunningApplyShouldOverwriteOwnTrackedFiles(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	var copiedFiles [][2]string

	fs := &utils.MoqFileSystem{
		PathExistsFunc:       pathExistsNoVariants,
		ReadFileContentsFunc: readFileByName,
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			switch path {
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree":
				return []os.DirEntry{&fakeDirEntry{name: "skills", isDir: true}}, nil
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree/skills":
				return []os.DirEntry{&fakeDirEntry{name: "my-skill", isDir: true}}, nil
			default:
				return nil, nil
			}
		},
		CreateDirectoryFunc: func(path string) error { return nil },
		CopyFileFunc: func(src, dst string) (int64, error) {
			copiedFiles = append(copiedFiles, [2]string{src, dst})
			return 100, nil
		},
	}

	git := &workspace.MoqGit{
		WorktreeAddFunc: func(ctx context.Context, repoDir, ref string) (string, error) {
			return repoDir + "/tempdir-worktree", nil
		},
	}

	store := &state.MoqBlockStore{
		// Path is tracked by the same subscription and agent — update is safe.
		QueryByPathFunc: func(ctx context.Context, path string) (*state.Record, error) {
			return &state.Record{
				Path:         path,
				Subscription: "acme-platform",
				Agent:        "claude-code",
			}, nil
		},
		QueryBySubscriptionAgentFunc: func(ctx context.Context, subscription, agent string) ([]state.Record, error) {
			return nil, nil
		},
		RecordBatchFunc: func(ctx context.Context, records []state.Record) error {
			return nil
		},
	}

	exporterResp := &exporter.ApplyResponse{
		Results: []exporter.BlockResult{
			{
				Name: "my-skill",
				Placements: []exporter.Placement{
					{
						Path:   "/output/.claude/skills/my-skill.md",
						Source: "skills/my-skill/my-skill.md",
					},
				},
			},
		},
	}
	dispatcher := &exporter.MoqDispatcher{
		ApplyFunc: func(ctx context.Context, agent string, req *exporter.ApplyRequest) (*exporter.ApplyResponse, error) {
			return exporterResp, nil
		},
	}

	deps := makeDeps(fs, store, git, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.NoError(t, err)
	require.Len(t, copiedFiles, 1, "own tracked file should be overwritten")
}

func TestRun_RunningApplyShouldCleanUpOrphanedFiles(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	var removedPaths []string

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			if hasSuffix(path, "variants.yaml") {
				return false, nil
			}

			if path == "/output/.claude/skills/new-skill.md" {
				return false, nil
			}

			return true, nil
		},
		ReadFileContentsFunc: readFileByName,
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			switch path {
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree":
				return []os.DirEntry{&fakeDirEntry{name: "skills", isDir: true}}, nil
			case "/home/user/.cache/cova/repos/github.com/acme/platform/tempdir-worktree/skills":
				return []os.DirEntry{&fakeDirEntry{name: "new-skill", isDir: true}}, nil
			default:
				return nil, nil
			}
		},
		CreateDirectoryFunc: func(path string) error { return nil },
		CopyFileFunc:        func(src, dst string) (int64, error) { return 100, nil },
		RemovePathFunc: func(path string) error {
			removedPaths = append(removedPaths, path)
			return nil
		},
	}

	git := &workspace.MoqGit{
		WorktreeAddFunc: func(ctx context.Context, repoDir, ref string) (string, error) {
			return repoDir + "/tempdir-worktree", nil
		},
	}

	store := &state.MoqBlockStore{
		QueryByPathFunc: func(ctx context.Context, path string) (*state.Record, error) {
			return nil, state.ErrNotFound
		},
		// Previous state includes an old orphaned file.
		QueryBySubscriptionAgentFunc: func(ctx context.Context, subscription, agent string) ([]state.Record, error) {
			return []state.Record{
				{
					Path:         "/output/.claude/skills/old-skill.md",
					Subscription: "acme-platform",
					Agent:        "claude-code",
				},
			}, nil
		},
		RecordBatchFunc: func(ctx context.Context, records []state.Record) error {
			return nil
		},
		DeleteByPathsFunc: func(ctx context.Context, paths []string) error {
			return nil
		},
	}

	exporterResp := &exporter.ApplyResponse{
		Results: []exporter.BlockResult{
			{
				Name: "new-skill",
				Placements: []exporter.Placement{
					{
						Path:   "/output/.claude/skills/new-skill.md",
						Source: "skills/new-skill/new-skill.md",
					},
				},
			},
		},
	}
	dispatcher := &exporter.MoqDispatcher{
		ApplyFunc: func(ctx context.Context, agent string, req *exporter.ApplyRequest) (*exporter.ApplyResponse, error) {
			return exporterResp, nil
		},
	}

	deps := makeDeps(fs, store, git, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.NoError(t, err)
	require.Contains(t, removedPaths, "/output/.claude/skills/old-skill.md")
	require.Len(t, store.DeleteByPathsCalls(), 1)
	require.Contains(t, store.DeleteByPathsCalls()[0].Paths, "/output/.claude/skills/old-skill.md")
}

func TestRun_RunningApplyShouldApplyOnlyNamedSubscriptions(t *testing.T) {
	ctx := t.Context()

	const twoSubsConfig = `
agents:
  - claude-code
subscriptions:
  - name: acme-platform
    repo: https://github.com/acme/platform.git
  - name: acme-tools
    repo: https://github.com/acme/tools.git
`

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	var worktreeAddCalls []string

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte(twoSubsConfig), nil
		},
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) { return nil, nil },
	}

	git := &workspace.MoqGit{
		WorktreeAddFunc: func(ctx context.Context, repoDir, ref string) (string, error) {
			worktreeAddCalls = append(worktreeAddCalls, repoDir)
			return repoDir + "/tempdir-worktree", nil
		},
	}

	store := &state.MoqBlockStore{
		QueryBySubscriptionAgentFunc: func(ctx context.Context, subscription, agent string) ([]state.Record, error) {
			return nil, nil
		},
		RecordBatchFunc: func(ctx context.Context, records []state.Record) error { return nil },
	}

	dispatcher := &exporter.MoqDispatcher{
		ApplyFunc: func(ctx context.Context, agent string, req *exporter.ApplyRequest) (*exporter.ApplyResponse, error) {
			return &exporter.ApplyResponse{}, nil
		},
	}

	deps := makeDeps(fs, store, git, dispatcher, envMgr, userMgr)

	err := Run(ctx, deps, []string{"acme-platform"})

	require.NoError(t, err)
	// Only the first subscription's workspace should be visited.
	require.Len(t, worktreeAddCalls, 1)
	require.Contains(t, worktreeAddCalls[0], "github.com/acme/platform")
}

func TestRun_RunningApplyShouldReturnErrorWhenConfigLoadFails(t *testing.T) {
	ctx := t.Context()

	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return nil, errors.New("disk read error")
		},
	}
	deps := makeDeps(fs, nil, nil, nil, envMgr, userMgr)

	err := Run(ctx, deps, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "loading config")
}

// fakeDirEntry is a minimal os.DirEntry for use in tests.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f *fakeDirEntry) Name() string               { return f.name }
func (f *fakeDirEntry) IsDir() bool                { return f.isDir }
func (f *fakeDirEntry) Type() os.FileMode          { return 0 }
func (f *fakeDirEntry) Info() (os.FileInfo, error) { return nil, errors.New("not implemented") }
