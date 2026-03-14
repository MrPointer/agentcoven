package workspace

import (
	"context"
	"fmt"
	"strings"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// Git defines the operations needed for managing git repositories.
type Git interface {
	// Clone clones a repository from the given URL into the target directory.
	Clone(ctx context.Context, repoURL, targetDir string) error

	// Fetch fetches the latest changes in the given repository directory.
	Fetch(ctx context.Context, repoDir string) error

	// RevParse resolves a ref to a commit hash in the given repository directory.
	RevParse(ctx context.Context, repoDir, ref string) (string, error)

	// Checkout checks out the given ref in the repository directory.
	Checkout(ctx context.Context, repoDir, ref string) error

	// WorktreeAdd creates a worktree for the given ref under the workspace cache directory.
	// If ref is empty, the worktree is created at HEAD.
	// It returns the path to the created worktree directory.
	WorktreeAdd(ctx context.Context, repoDir, ref string) (string, error)

	// WorktreeRemove removes the worktree at the given path.
	// It is idempotent — removing an already-removed worktree does not error.
	WorktreeRemove(ctx context.Context, worktreePath string) error
}

// DefaultGit implements Git using Commander and FileSystem.
type DefaultGit struct {
	commander utils.Commander
	fs        utils.FileSystem
}

var _ Git = (*DefaultGit)(nil)

// NewDefaultGit creates a new DefaultGit with the given Commander and FileSystem.
func NewDefaultGit(commander utils.Commander, fs utils.FileSystem) *DefaultGit {
	return &DefaultGit{commander: commander, fs: fs}
}

// Clone clones a repository from the given URL into the target directory.
func (g *DefaultGit) Clone(ctx context.Context, repoURL, targetDir string) error {
	_, err := g.commander.RunCommand(
		ctx,
		"git",
		[]string{"clone", repoURL, targetDir},
		utils.WithCaptureOutput(),
	)
	if err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	return nil
}

// Fetch fetches the latest changes in the given repository directory.
func (g *DefaultGit) Fetch(ctx context.Context, repoDir string) error {
	_, err := g.commander.RunCommand(
		ctx,
		"git",
		[]string{"fetch", "--all"},
		utils.WithDir(repoDir),
		utils.WithCaptureOutput(),
	)
	if err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	return nil
}

// RevParse resolves a ref to a commit hash in the given repository directory.
func (g *DefaultGit) RevParse(ctx context.Context, repoDir, ref string) (string, error) {
	result, err := g.commander.RunCommand(
		ctx,
		"git",
		[]string{"rev-parse", "--verify", ref},
		utils.WithDir(repoDir),
		utils.WithCaptureOutput(),
	)
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed for ref %q: %w", ref, err)
	}

	return strings.TrimSpace(string(result.Stdout)), nil
}

// Checkout checks out the given ref in the repository directory.
func (g *DefaultGit) Checkout(ctx context.Context, repoDir, ref string) error {
	_, err := g.commander.RunCommand(
		ctx,
		"git",
		[]string{"checkout", ref},
		utils.WithDir(repoDir),
		utils.WithCaptureOutput(),
	)
	if err != nil {
		return fmt.Errorf("git checkout failed for ref %q: %w", ref, err)
	}

	return nil
}

// WorktreeAdd creates a worktree in detached HEAD mode for the given ref under the workspace cache directory.
// If ref is empty, HEAD is used. It returns the path to the created worktree directory.
func (g *DefaultGit) WorktreeAdd(ctx context.Context, repoDir, ref string) (string, error) {
	worktreeRef := ref
	if worktreeRef == "" {
		worktreeRef = "HEAD"
	}

	worktreePath, err := g.fs.CreateTemporaryDirectory(repoDir)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree directory: %w", err)
	}

	_, err = g.commander.RunCommand(
		ctx,
		"git",
		[]string{"worktree", "add", "--detach", worktreePath, worktreeRef},
		utils.WithDir(repoDir),
		utils.WithCaptureOutput(),
	)
	if err != nil {
		return "", fmt.Errorf("git worktree add failed for ref %q: %w", ref, err)
	}

	return worktreePath, nil
}

// WorktreeRemove removes the worktree at the given path.
// It is idempotent — if the path does not exist it returns without error.
func (g *DefaultGit) WorktreeRemove(ctx context.Context, worktreePath string) error {
	exists, err := g.fs.PathExists(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to check worktree path: %w", err)
	}

	if !exists {
		return nil
	}

	_, err = g.commander.RunCommand(
		ctx,
		"git",
		[]string{"worktree", "remove", "--force", worktreePath},
		utils.WithCaptureOutput(),
	)
	if err != nil {
		return fmt.Errorf("git worktree remove failed: %w", err)
	}

	stillExists, err := g.fs.PathExists(worktreePath)
	if err != nil {
		return fmt.Errorf("failed to check worktree path after removal: %w", err)
	}

	if stillExists {
		if removeErr := g.fs.RemovePath(worktreePath); removeErr != nil {
			return fmt.Errorf("failed to remove worktree directory: %w", removeErr)
		}
	}

	return nil
}
