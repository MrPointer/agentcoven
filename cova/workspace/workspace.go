// Package workspace manages local git clones of coven repositories under the XDG cache directory.
package workspace

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

// DefaultBasePath returns the base path for cova workspace repositories.
// It uses $XDG_CACHE_HOME/cova/repos/ if set, otherwise ~/.cache/cova/repos/.
func DefaultBasePath(
	envManager osmanager.EnvironmentManager,
	userManager osmanager.UserManager,
) (string, error) {
	cacheHome := envManager.Getenv("XDG_CACHE_HOME")
	if cacheHome == "" {
		homeDir, err := userManager.GetHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}

		cacheHome = filepath.Join(homeDir, ".cache")
	}

	return filepath.Join(cacheHome, "cova", "repos"), nil
}

// NormalizeURL normalizes a repository URL to a canonical form for consistent
// workspace directory naming. Supported formats:
//   - HTTPS: https://github.com/acme/blocks.git -> github.com/acme/blocks
//   - HTTP:  http://github.com/acme/blocks.git  -> github.com/acme/blocks
//   - SSH:   git@github.com:acme/blocks.git     -> github.com/acme/blocks
//   - file:  file:///tmp/repo                    -> /tmp/repo
func NormalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("empty repository URL")
	}

	// Handle SSH format: git@host:path
	if strings.HasPrefix(rawURL, "git@") {
		return normalizeSSH(rawURL)
	}

	// Handle file:// URLs
	if strings.HasPrefix(rawURL, "file://") {
		return normalizeFileURL(rawURL)
	}

	// Handle HTTPS/HTTP URLs
	if strings.HasPrefix(rawURL, "https://") || strings.HasPrefix(rawURL, "http://") {
		return normalizeHTTPURL(rawURL)
	}

	return "", fmt.Errorf("unsupported URL format: %s", rawURL)
}

func normalizeSSH(rawURL string) (string, error) {
	// git@host:path -> host/path
	withoutPrefix := strings.TrimPrefix(rawURL, "git@")

	before, after, ok := strings.Cut(withoutPrefix, ":")
	if !ok {
		return "", fmt.Errorf("invalid SSH URL (missing ':'): %s", rawURL)
	}

	host := strings.ToLower(before)
	path := after
	path = strings.TrimRight(path, "/")
	path = stripGitSuffix(path)

	return host + "/" + path, nil
}

func normalizeFileURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid file URL: %w", err)
	}

	path := parsed.Path
	path = strings.TrimRight(path, "/")

	return path, nil
}

func normalizeHTTPURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	host := strings.ToLower(parsed.Hostname())
	path := parsed.Path
	path = strings.TrimRight(path, "/")
	path = stripGitSuffix(path)

	return host + path, nil
}

func stripGitSuffix(s string) string {
	return strings.TrimSuffix(s, ".git")
}

// Ensure ensures a workspace exists for the given repository URL.
// If the workspace does not exist, it clones the repository.
// If it already exists, it fetches the latest changes.
// If ref is non-empty, it checks out that ref after ensuring.
// Returns the workspace directory path.
func Ensure(
	ctx context.Context,
	git Git,
	fs utils.FileSystem,
	basePath string,
	repoURL string,
	ref string,
) (string, error) {
	normalized, err := NormalizeURL(repoURL)
	if err != nil {
		return "", fmt.Errorf("failed to normalize URL: %w", err)
	}

	workspacePath := filepath.Join(basePath, normalized)

	exists, err := fs.PathExists(workspacePath)
	if err != nil {
		return "", fmt.Errorf("failed to check workspace path: %w", err)
	}

	if exists {
		if err := git.Fetch(ctx, workspacePath); err != nil {
			return "", err
		}
	} else {
		if err := fs.CreateDirectory(filepath.Dir(workspacePath)); err != nil {
			return "", fmt.Errorf("failed to create workspace parent directory: %w", err)
		}

		if err := git.Clone(ctx, repoURL, workspacePath); err != nil {
			return "", err
		}
	}

	if ref != "" {
		if err := git.Checkout(ctx, workspacePath, ref); err != nil {
			return "", err
		}
	}

	return workspacePath, nil
}
