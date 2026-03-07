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
}

// DefaultGit implements Git using Commander.
type DefaultGit struct {
	commander utils.Commander
}

var _ Git = (*DefaultGit)(nil)

// NewDefaultGit creates a new DefaultGit with the given Commander.
func NewDefaultGit(commander utils.Commander) *DefaultGit {
	return &DefaultGit{commander: commander}
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
