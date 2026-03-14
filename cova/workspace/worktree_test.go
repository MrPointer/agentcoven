package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/workspace"
)

func TestDefaultGit_AddingWorktreeShouldCreateWorktreeAtRef(t *testing.T) {
	fs := &utils.MoqFileSystem{
		CreateTemporaryDirectoryFunc: func(dir string) (string, error) {
			require.Equal(t, "/repo/path", dir)
			return "/repo/path/tempdir-abc", nil
		},
	}
	commander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			require.Equal(t, "git", name)
			require.Equal(t, []string{"worktree", "add", "--detach", "/repo/path/tempdir-abc", "v1.2.3"}, args)

			return &utils.Result{}, nil
		},
	}
	git := workspace.NewDefaultGit(commander, fs)

	path, err := git.WorktreeAdd(t.Context(), "/repo/path", "v1.2.3")

	require.NoError(t, err)
	require.Equal(t, "/repo/path/tempdir-abc", path)
	require.Len(t, commander.RunCommandCalls(), 1)
}

func TestDefaultGit_AddingWorktreeShouldUseHEADWhenRefIsEmpty(t *testing.T) {
	fs := &utils.MoqFileSystem{
		CreateTemporaryDirectoryFunc: func(dir string) (string, error) {
			return "/repo/path/tempdir-xyz", nil
		},
	}
	commander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			require.Equal(t, []string{"worktree", "add", "--detach", "/repo/path/tempdir-xyz", "HEAD"}, args)
			return &utils.Result{}, nil
		},
	}
	git := workspace.NewDefaultGit(commander, fs)

	path, err := git.WorktreeAdd(t.Context(), "/repo/path", "")

	require.NoError(t, err)
	require.Equal(t, "/repo/path/tempdir-xyz", path)
}

func TestDefaultGit_AddingWorktreeShouldReturnErrorWhenDirectoryCreationFails(t *testing.T) {
	fs := &utils.MoqFileSystem{
		CreateTemporaryDirectoryFunc: func(dir string) (string, error) {
			return "", errors.New("disk full")
		},
	}
	commander := &utils.MoqCommander{}
	git := workspace.NewDefaultGit(commander, fs)

	_, err := git.WorktreeAdd(t.Context(), "/repo/path", "v1.0.0")

	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create worktree directory")
	require.Empty(t, commander.RunCommandCalls())
}

func TestDefaultGit_AddingWorktreeShouldReturnErrorWhenCommandFails(t *testing.T) {
	fs := &utils.MoqFileSystem{
		CreateTemporaryDirectoryFunc: func(dir string) (string, error) {
			return "/repo/path/tempdir-abc", nil
		},
	}
	commander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return nil, errors.New("invalid ref")
		},
	}
	git := workspace.NewDefaultGit(commander, fs)

	_, err := git.WorktreeAdd(t.Context(), "/repo/path", "bad-ref")

	require.Error(t, err)
	require.Contains(t, err.Error(), "git worktree add failed")
}

func TestDefaultGit_RemovingWorktreeShouldRemoveWorktreeAtPath(t *testing.T) {
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			if path == "/worktrees/abc" {
				return true, nil
			}

			return false, nil
		},
		RemovePathFunc: func(path string) error {
			return nil
		},
	}
	commander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			require.Equal(t, "git", name)
			require.Equal(t, []string{"worktree", "remove", "--force", "/worktrees/abc"}, args)

			return &utils.Result{}, nil
		},
	}
	git := workspace.NewDefaultGit(commander, fs)

	err := git.WorktreeRemove(t.Context(), "/worktrees/abc")

	require.NoError(t, err)
	require.Len(t, commander.RunCommandCalls(), 1)
}

func TestDefaultGit_RemovingWorktreeShouldBeIdempotentWhenPathDoesNotExist(t *testing.T) {
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, nil
		},
	}
	commander := &utils.MoqCommander{}
	git := workspace.NewDefaultGit(commander, fs)

	err := git.WorktreeRemove(t.Context(), "/worktrees/already-gone")

	require.NoError(t, err)
	require.Empty(t, commander.RunCommandCalls())
}

func TestDefaultGit_RemovingWorktreeShouldReturnErrorWhenCommandFails(t *testing.T) {
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
	}
	commander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return nil, errors.New("locked worktree")
		},
	}
	git := workspace.NewDefaultGit(commander, fs)

	err := git.WorktreeRemove(t.Context(), "/worktrees/abc")

	require.Error(t, err)
	require.Contains(t, err.Error(), "git worktree remove failed")
}

func TestDefaultGit_RemovingWorktreeShouldCleanUpDirectoryWhenGitDoesNotRemoveIt(t *testing.T) {
	callCount := 0
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			callCount++
			// First call (pre-check): exists. Second call (post-remove): still exists.
			return true, nil
		},
		RemovePathFunc: func(path string) error {
			require.Equal(t, "/worktrees/abc", path)
			return nil
		},
	}
	commander := &utils.MoqCommander{
		RunCommandFunc: func(ctx context.Context, name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return &utils.Result{}, nil
		},
	}
	git := workspace.NewDefaultGit(commander, fs)

	err := git.WorktreeRemove(t.Context(), "/worktrees/abc")

	require.NoError(t, err)
	require.Len(t, fs.RemovePathCalls(), 1)
}
