package block_test

import (
	"errors"
	"io/fs"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/block"
	"github.com/MrPointer/agentcoven/cova/utils"
)

// fakeDirEntry implements os.DirEntry for test purposes.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string      { return f.name }
func (f fakeDirEntry) IsDir() bool       { return f.isDir }
func (f fakeDirEntry) Type() fs.FileMode { return 0 }
func (f fakeDirEntry) Info() (fs.FileInfo, error) {
	return fakeFileInfo(f), nil
}

type fakeFileInfo struct {
	name  string
	isDir bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.isDir }
func (f fakeFileInfo) Sys() any           { return nil }

// dirEntry and fileEntry are convenience constructors for test fixtures.
func dirEntry(name string) fakeDirEntry  { return fakeDirEntry{name: name, isDir: true} }
func fileEntry(name string) fakeDirEntry { return fakeDirEntry{name: name, isDir: false} }

func TestDiscover_DiscoveringShouldReturnEmptyMapWhenCovenRootHasNoTypeDirs(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			return []os.DirEntry{
				fileEntry("manifest.yaml"),
			}, nil
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.NoError(t, err)
	require.Empty(t, result)
}

func TestDiscover_DiscoveringShouldSkipDotPrefixedDirectories(t *testing.T) {
	calls := map[string]bool{}

	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			calls[path] = true

			if path == "/coven" {
				return []os.DirEntry{
					dirEntry(".git"),
					dirEntry(".github"),
					dirEntry("skills"),
				}, nil
			}

			return []os.DirEntry{
				dirEntry("acme-test"),
			}, nil
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.NoError(t, err)
	require.False(t, calls["/coven/.git"], ".git should not be scanned")
	require.False(t, calls["/coven/.github"], ".github should not be scanned")
	require.Contains(t, result, "skills")
}

func TestDiscover_DiscoveringShouldIgnoreNonDirectoryEntriesAtCovenRoot(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			if path == "/coven" {
				return []os.DirEntry{
					fileEntry("manifest.yaml"),
					fileEntry("README.md"),
					dirEntry("skills"),
				}, nil
			}

			return []os.DirEntry{
				dirEntry("acme-platform-code-review"),
			}, nil
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Contains(t, result, "skills")
}

func TestDiscover_DiscoveringShouldIgnoreNonDirectoryEntriesWithinTypeDirectories(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			if path == "/coven" {
				return []os.DirEntry{
					dirEntry("skills"),
				}, nil
			}

			return []os.DirEntry{
				fileEntry("some-file.txt"),
				dirEntry("acme-platform-code-review"),
			}, nil
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.NoError(t, err)
	require.Len(t, result["skills"], 1)
	require.Equal(t, "acme-platform-code-review", result["skills"][0].Name)
}

func TestDiscover_DiscoveringShouldReturnBlocksGroupedByType(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			switch path {
			case "/coven":
				return []os.DirEntry{
					dirEntry("skills"),
					dirEntry("agents"),
					dirEntry("rules"),
				}, nil
			case "/coven/skills":
				return []os.DirEntry{
					dirEntry("acme-platform-code-review"),
					dirEntry("acme-platform-testing"),
				}, nil
			case "/coven/agents":
				return []os.DirEntry{
					dirEntry("acme-platform-reviewer"),
				}, nil
			case "/coven/rules":
				return []os.DirEntry{}, nil
			default:
				return nil, errors.New("unexpected path: " + path)
			}
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.NoError(t, err)
	require.Len(t, result["skills"], 2)
	require.Len(t, result["agents"], 1)
	require.Empty(t, result["rules"])

	skillNames := []string{result["skills"][0].Name, result["skills"][1].Name}
	require.ElementsMatch(t, []string{"acme-platform-code-review", "acme-platform-testing"}, skillNames)

	require.Equal(t, "acme-platform-reviewer", result["agents"][0].Name)
	require.Equal(t, "agents", result["agents"][0].Type)
	require.Equal(t, "agents/acme-platform-reviewer", result["agents"][0].SourceDir)
}

func TestDiscover_DiscoveringShouldHandleCustomBlockTypes(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			if path == "/coven" {
				return []os.DirEntry{
					dirEntry("custom-type"),
				}, nil
			}

			return []os.DirEntry{
				dirEntry("my-custom-block"),
			}, nil
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.NoError(t, err)
	require.Contains(t, result, "custom-type")
	require.Len(t, result["custom-type"], 1)
	require.Equal(t, "my-custom-block", result["custom-type"][0].Name)
	require.Equal(t, "custom-type/my-custom-block", result["custom-type"][0].SourceDir)
}

func TestDiscover_DiscoveringShouldReturnErrorWhenCovenRootCannotBeRead(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			return nil, errors.New("permission denied")
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "reading coven root")
}

func TestDiscover_DiscoveringShouldReturnErrorWhenTypeDirectoryCannotBeRead(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			if path == "/coven" {
				return []os.DirEntry{
					dirEntry("skills"),
				}, nil
			}

			return nil, errors.New("permission denied")
		},
	}

	result, err := block.Discover(mockFS, "/coven")

	require.Error(t, err)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "reading type directory")
}

func TestResolveVariant_ResolvingShouldReturnOriginalSourceDirWhenNoVariantsFile(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, nil
		},
	}

	resolved, include, err := block.ResolveVariant(mockFS, "/coven", "skills/acme-platform-code-review", "claude-code")

	require.NoError(t, err)
	require.True(t, include)
	require.Equal(t, "skills/acme-platform-code-review", resolved)
}

func TestResolveVariant_ResolvingShouldReturnVariantSubdirWhenExporterIsListed(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte("variants:\n  - claude-code\n  - opencode\n"), nil
		},
	}

	resolved, include, err := block.ResolveVariant(
		mockFS, "/coven", "skills/acme-platform-deploy-pipeline", "claude-code",
	)

	require.NoError(t, err)
	require.True(t, include)
	require.Equal(t, "skills/acme-platform-deploy-pipeline/claude-code", resolved)
}

func TestResolveVariant_ResolvingShouldSignalSkipWhenExporterIsNotListed(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte("variants:\n  - claude-code\n  - opencode\n"), nil
		},
	}

	resolved, include, err := block.ResolveVariant(
		mockFS, "/coven", "skills/acme-platform-deploy-pipeline", "cursor",
	)

	require.NoError(t, err)
	require.False(t, include)
	require.Empty(t, resolved)
}

func TestResolveVariant_ResolvingShouldReturnErrorWhenPathExistsFails(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return false, errors.New("stat error")
		},
	}

	resolved, include, err := block.ResolveVariant(mockFS, "/coven", "skills/acme-block", "claude-code")

	require.Error(t, err)
	require.False(t, include)
	require.Empty(t, resolved)
	require.Contains(t, err.Error(), "checking variants file")
}

func TestResolveVariant_ResolvingShouldReturnErrorWhenVariantsFileCannotBeRead(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return nil, errors.New("read error")
		},
	}

	resolved, include, err := block.ResolveVariant(mockFS, "/coven", "skills/acme-block", "claude-code")

	require.Error(t, err)
	require.False(t, include)
	require.Empty(t, resolved)
	require.Contains(t, err.Error(), "reading variants file")
}

func TestResolveVariant_ResolvingShouldReturnErrorWhenVariantsFileIsMalformed(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) {
			return true, nil
		},
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return []byte("variants: [invalid yaml"), nil
		},
	}

	resolved, include, err := block.ResolveVariant(mockFS, "/coven", "skills/acme-block", "claude-code")

	require.Error(t, err)
	require.False(t, include)
	require.Empty(t, resolved)
	require.Contains(t, err.Error(), "parsing variants file")
}
