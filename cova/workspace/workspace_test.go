package workspace_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
	"github.com/stretchr/testify/require"
)

// --- DefaultBasePath tests ---

func TestDefaultBasePath_ResolvingBasePathShouldUseXDGCacheHomeWhenSet(t *testing.T) {
	envManager := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			if key == "XDG_CACHE_HOME" {
				return "/custom/cache"
			}

			return ""
		},
	}
	userManager := &osmanager.MoqUserManager{}

	result, err := workspace.DefaultBasePath(envManager, userManager)

	require.NoError(t, err)
	require.Equal(t, "/custom/cache/cova/repos", result)
}

func TestDefaultBasePath_ResolvingBasePathShouldFallBackToHomeDirWhenXDGNotSet(t *testing.T) {
	envManager := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			return ""
		},
	}
	userManager := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) {
			return "/home/testuser", nil
		},
	}

	result, err := workspace.DefaultBasePath(envManager, userManager)

	require.NoError(t, err)
	require.Equal(t, "/home/testuser/.cache/cova/repos", result)
}

func TestDefaultBasePath_ResolvingBasePathShouldReturnErrorWhenHomeDirFails(t *testing.T) {
	envManager := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string {
			return ""
		},
	}
	userManager := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) {
			return "", errors.New("no home dir")
		},
	}

	_, err := workspace.DefaultBasePath(envManager, userManager)

	require.Error(t, err)
	require.Contains(t, err.Error(), "home directory")
}

// --- NormalizeURL tests ---

func TestNormalizeURL_NormalizingURLShouldSucceed(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"WhenInputIsHTTPSWithGitSuffix", "https://github.com/acme/blocks.git", "github.com/acme/blocks"},
		{"WhenInputIsHTTPSWithoutGitSuffix", "https://github.com/acme/blocks", "github.com/acme/blocks"},
		{"WhenInputIsHTTPSWithTrailingSlash", "https://github.com/acme/blocks/", "github.com/acme/blocks"},
		{
			"WhenInputIsHTTPSWithGitSuffixAndTrailingSlash",
			"https://github.com/acme/blocks.git/",
			"github.com/acme/blocks",
		},
		{"WhenInputIsHTTPWithGitSuffix", "http://github.com/acme/blocks.git", "github.com/acme/blocks"},
		{"WhenInputIsHTTPSWithMixedCaseHost", "https://GitHub.COM/acme/blocks.git", "github.com/acme/blocks"},
		{"WhenInputIsSSH", "git@github.com:acme/blocks.git", "github.com/acme/blocks"},
		{"WhenInputIsSSHWithoutGitSuffix", "git@github.com:acme/blocks", "github.com/acme/blocks"},
		{"WhenInputIsSSHWithMixedCaseHost", "git@GitHub.COM:acme/blocks.git", "github.com/acme/blocks"},
		{"WhenInputIsSSHWithTrailingSlash", "git@github.com:acme/blocks.git/", "github.com/acme/blocks"},
		{"WhenInputIsFileURL", "file:///tmp/repo", "/tmp/repo"},
		{"WhenInputIsFileURLWithTrailingSlash", "file:///tmp/repo/", "/tmp/repo"},
		{"WhenInputHasLeadingWhitespace", "  https://github.com/acme/blocks.git  ", "github.com/acme/blocks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := workspace.NormalizeURL(tt.input)

			require.NoError(t, err)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeURL_NormalizingURLShouldReturnError(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"WhenInputIsEmpty", ""},
		{"WhenInputIsWhitespaceOnly", "   "},
		{"WhenInputIsUnsupportedFormat", "ftp://example.com/repo"},
		{"WhenInputIsSSHWithoutColon", "git@github.com/acme/blocks"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := workspace.NormalizeURL(tt.input)

			require.Error(t, err)
		})
	}
}

// --- DefaultGit tests ---

func TestDefaultGit_CloningRepoShouldCallGitClone(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			require.Equal(t, "git", name)
			require.Equal(t, []string{"clone", "https://github.com/acme/blocks.git", "/workspace/path"}, args)

			return &utils.Result{}, nil
		},
	}
	git := workspace.NewDefaultGit(commander)

	err := git.Clone("https://github.com/acme/blocks.git", "/workspace/path")

	require.NoError(t, err)
	require.Len(t, commander.RunCommandCalls(), 1)
}

func TestDefaultGit_CloningRepoShouldReturnErrorWhenCommandFails(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return nil, errors.New("clone failed")
		},
	}
	git := workspace.NewDefaultGit(commander)

	err := git.Clone("https://github.com/acme/blocks.git", "/workspace/path")

	require.Error(t, err)
	require.Contains(t, err.Error(), "git clone failed")
}

func TestDefaultGit_FetchingRepoShouldCallGitFetch(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			require.Equal(t, "git", name)
			require.Equal(t, []string{"fetch", "--all"}, args)

			return &utils.Result{}, nil
		},
	}
	git := workspace.NewDefaultGit(commander)

	err := git.Fetch("/workspace/path")

	require.NoError(t, err)
	require.Len(t, commander.RunCommandCalls(), 1)
}

func TestDefaultGit_FetchingRepoShouldReturnErrorWhenCommandFails(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return nil, errors.New("fetch failed")
		},
	}
	git := workspace.NewDefaultGit(commander)

	err := git.Fetch("/workspace/path")

	require.Error(t, err)
	require.Contains(t, err.Error(), "git fetch failed")
}

func TestDefaultGit_RevParsingRefShouldReturnCommitHash(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			require.Equal(t, "git", name)
			require.Equal(t, []string{"rev-parse", "--verify", "v1.0.0"}, args)

			return &utils.Result{Stdout: []byte("abc123\n")}, nil
		},
	}
	git := workspace.NewDefaultGit(commander)

	hash, err := git.RevParse("/workspace/path", "v1.0.0")

	require.NoError(t, err)
	require.Equal(t, "abc123", hash)
}

func TestDefaultGit_RevParsingRefShouldReturnErrorWhenCommandFails(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return nil, errors.New("unknown ref")
		},
	}
	git := workspace.NewDefaultGit(commander)

	_, err := git.RevParse("/workspace/path", "nonexistent")

	require.Error(t, err)
	require.Contains(t, err.Error(), "rev-parse failed")
}

func TestDefaultGit_CheckingOutRefShouldCallGitCheckout(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			require.Equal(t, "git", name)
			require.Equal(t, []string{"checkout", "v1.0.0"}, args)

			return &utils.Result{}, nil
		},
	}
	git := workspace.NewDefaultGit(commander)

	err := git.Checkout("/workspace/path", "v1.0.0")

	require.NoError(t, err)
	require.Len(t, commander.RunCommandCalls(), 1)
}

func TestDefaultGit_CheckingOutRefShouldReturnErrorWhenCommandFails(t *testing.T) {
	commander := &utils.MoqCommander{
		RunCommandFunc: func(name string, args []string, opts ...utils.Option) (*utils.Result, error) {
			return nil, errors.New("checkout failed")
		},
	}
	git := workspace.NewDefaultGit(commander)

	err := git.Checkout("/workspace/path", "v1.0.0")

	require.Error(t, err)
	require.Contains(t, err.Error(), "checkout failed")
}

// --- Ensure tests ---

func TestEnsure_EnsuringWorkspaceShouldCloneWhenNotExists(t *testing.T) {
	git := &workspace.MoqGit{
		CloneFunc: func(repoURL, targetDir string) error {
			require.Equal(t, "https://github.com/acme/blocks.git", repoURL)
			require.Equal(t, "/cache/cova/repos/github.com/acme/blocks", targetDir)

			return nil
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, nil
		},
		CreateDirectoryFunc: func(path string) error {
			return nil
		},
	}

	result, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "")

	require.NoError(t, err)
	require.Equal(t, "/cache/cova/repos/github.com/acme/blocks", result)
	require.Len(t, git.CloneCalls(), 1)
	require.Empty(t, git.FetchCalls())
}

func TestEnsure_EnsuringWorkspaceShouldFetchWhenAlreadyExists(t *testing.T) {
	git := &workspace.MoqGit{
		FetchFunc: func(repoDir string) error {
			require.Equal(t, "/cache/cova/repos/github.com/acme/blocks", repoDir)
			return nil
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
	}

	result, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "")

	require.NoError(t, err)
	require.Equal(t, "/cache/cova/repos/github.com/acme/blocks", result)
	require.Len(t, git.FetchCalls(), 1)
	require.Empty(t, git.CloneCalls())
}

func TestEnsure_EnsuringWorkspaceShouldCheckoutRefWhenSpecified(t *testing.T) {
	git := &workspace.MoqGit{
		FetchFunc: func(repoDir string) error {
			return nil
		},
		CheckoutFunc: func(repoDir, ref string) error {
			require.Equal(t, "/cache/cova/repos/github.com/acme/blocks", repoDir)
			require.Equal(t, "v1.0.0", ref)

			return nil
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
	}

	result, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "v1.0.0")

	require.NoError(t, err)
	require.Equal(t, "/cache/cova/repos/github.com/acme/blocks", result)
	require.Len(t, git.CheckoutCalls(), 1)
}

func TestEnsure_EnsuringWorkspaceShouldNotCheckoutWhenRefIsEmpty(t *testing.T) {
	git := &workspace.MoqGit{
		FetchFunc: func(repoDir string) error {
			return nil
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
	}

	_, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "")

	require.NoError(t, err)
	require.Empty(t, git.CheckoutCalls())
}

func TestEnsure_EnsuringWorkspaceShouldReturnErrorWhenURLIsInvalid(t *testing.T) {
	git := &workspace.MoqGit{}
	fs := &utils.MoqFileSystem{}

	_, err := workspace.Ensure(git, fs, "/cache/cova/repos", "", "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "normalize URL")
}

func TestEnsure_EnsuringWorkspaceShouldReturnErrorWhenPathCheckFails(t *testing.T) {
	git := &workspace.MoqGit{}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, errors.New("disk error")
		},
	}

	_, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "workspace path")
}

func TestEnsure_EnsuringWorkspaceShouldReturnErrorWhenCloneFails(t *testing.T) {
	git := &workspace.MoqGit{
		CloneFunc: func(repoURL, targetDir string) error {
			return errors.New("git clone failed: network error")
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, nil
		},
		CreateDirectoryFunc: func(path string) error {
			return nil
		},
	}

	_, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "clone failed")
}

func TestEnsure_EnsuringWorkspaceShouldReturnErrorWhenFetchFails(t *testing.T) {
	git := &workspace.MoqGit{
		FetchFunc: func(repoDir string) error {
			return errors.New("git fetch failed: network error")
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
	}

	_, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "fetch failed")
}

func TestEnsure_EnsuringWorkspaceShouldReturnErrorWhenCreateDirectoryFails(t *testing.T) {
	git := &workspace.MoqGit{
		CloneFunc: func(repoURL, targetDir string) error {
			return nil
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, nil
		},
		CreateDirectoryFunc: func(path string) error {
			return errors.New("permission denied")
		},
	}

	_, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "")

	require.Error(t, err)
	require.Contains(t, err.Error(), "parent directory")
}

func TestEnsure_EnsuringWorkspaceShouldReturnErrorWhenCheckoutFails(t *testing.T) {
	git := &workspace.MoqGit{
		FetchFunc: func(repoDir string) error {
			return nil
		},
		CheckoutFunc: func(repoDir, ref string) error {
			return fmt.Errorf("git checkout failed for ref %q: bad ref", ref)
		},
	}
	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
	}

	_, err := workspace.Ensure(git, fs, "/cache/cova/repos", "https://github.com/acme/blocks.git", "bad-ref")

	require.Error(t, err)
	require.Contains(t, err.Error(), "checkout failed")
}
