package adapter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/utils"
)

func TestClaudeCodeAdapter_ApplyingSkilleBlockShouldProducePlacements(t *testing.T) {
	homeDir := "/home/testuser"
	workspace := "/workspace"

	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			return []os.DirEntry{
				fakeDirEntry{name: "SKILL.md", isDir: false},
				fakeDirEntry{name: "nested", isDir: true},
			}, nil
		},
	}

	a := newClaudeCodeAdapter(mockFS, homeDir)
	req := &ApplyRequest{
		Workspace: workspace,
		Blocks: map[string][]RequestBlock{
			"skills": {
				{Name: "acme-platform-code-review", Source: "skills/acme-platform-code-review"},
			},
		},
	}

	resp, err := a.apply(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)

	result := resp.Results[0]
	require.Equal(t, "acme-platform-code-review", result.Name)
	require.Nil(t, result.Error)
	require.Len(t, result.Placements, 1)
	require.Equal(
		t,
		filepath.Join(homeDir, ".claude", "skills", "acme-platform-code-review", "SKILL.md"),
		result.Placements[0].Path,
	)
	require.Equal(t, "skills/acme-platform-code-review/SKILL.md", result.Placements[0].Source)
}

func TestClaudeCodeAdapter_ApplyingAgentsBlockShouldProducePlacements(t *testing.T) {
	homeDir := "/home/testuser"
	workspace := "/workspace"

	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			return []os.DirEntry{
				fakeDirEntry{name: "agent.md", isDir: false},
			}, nil
		},
	}

	a := newClaudeCodeAdapter(mockFS, homeDir)
	req := &ApplyRequest{
		Workspace: workspace,
		Blocks: map[string][]RequestBlock{
			"agents": {
				{Name: "acme-platform-researcher", Source: "agents/acme-platform-researcher"},
			},
		},
	}

	resp, err := a.apply(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)

	result := resp.Results[0]
	require.Equal(t, "acme-platform-researcher", result.Name)
	require.Nil(t, result.Error)
	require.Len(t, result.Placements, 1)
	require.Equal(
		t,
		filepath.Join(homeDir, ".claude", "agents", "acme-platform-researcher", "agent.md"),
		result.Placements[0].Path,
	)
}

func TestClaudeCodeAdapter_ApplyingUnsupportedBlockTypeShouldReturnPerBlockError(t *testing.T) {
	homeDir := "/home/testuser"
	workspace := "/workspace"

	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			return nil, nil
		},
	}

	a := newClaudeCodeAdapter(mockFS, homeDir)
	req := &ApplyRequest{
		Workspace: workspace,
		Blocks: map[string][]RequestBlock{
			"rules": {
				{Name: "acme-platform-lint-rule", Source: "rules/acme-platform-lint-rule"},
			},
		},
	}

	resp, err := a.apply(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)

	result := resp.Results[0]
	require.Equal(t, "acme-platform-lint-rule", result.Name)
	require.NotNil(t, result.Error)
	require.Contains(t, *result.Error, "rules")
	require.Contains(t, *result.Error, "not supported")
	require.Nil(t, result.Placements)
}

func TestClaudeCodeAdapter_ApplyingMixedBlockTypesShouldHandleEachIndependently(t *testing.T) {
	homeDir := "/home/testuser"
	workspace := "/workspace"

	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			return []os.DirEntry{
				fakeDirEntry{name: "file.md", isDir: false},
			}, nil
		},
	}

	a := newClaudeCodeAdapter(mockFS, homeDir)
	req := &ApplyRequest{
		Workspace: workspace,
		Blocks: map[string][]RequestBlock{
			"skills": {
				{Name: "acme-skill", Source: "skills/acme-skill"},
			},
			"rules": {
				{Name: "acme-rule", Source: "rules/acme-rule"},
			},
		},
	}

	resp, err := a.apply(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, resp.Results, 2)

	var skillResult, ruleResult *BlockResult

	for i := range resp.Results {
		switch resp.Results[i].Name {
		case "acme-skill":
			skillResult = &resp.Results[i]
		case "acme-rule":
			ruleResult = &resp.Results[i]
		default:
			t.Fatalf("unexpected result name: %s", resp.Results[i].Name)
		}
	}

	require.NotNil(t, skillResult)
	require.Nil(t, skillResult.Error)
	require.Len(t, skillResult.Placements, 1)

	require.NotNil(t, ruleResult)
	require.NotNil(t, ruleResult.Error)
	require.Nil(t, ruleResult.Placements)
}

func TestClaudeCodeAdapter_ApplyingBlockWithNoFilesShouldReturnEmptyPlacements(t *testing.T) {
	mockFS := &utils.MoqFileSystem{
		ReadDirectoryFunc: func(path string) ([]os.DirEntry, error) {
			return []os.DirEntry{}, nil
		},
	}

	a := newClaudeCodeAdapter(mockFS, "/home/user")
	req := &ApplyRequest{
		Workspace: "/ws",
		Blocks: map[string][]RequestBlock{
			"skills": {
				{Name: "empty-skill", Source: "skills/empty-skill"},
			},
		},
	}

	resp, err := a.apply(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, resp.Results, 1)
	require.Nil(t, resp.Results[0].Error)
	require.Empty(t, resp.Results[0].Placements)
}

// fakeDirEntry is a minimal os.DirEntry implementation for tests.
type fakeDirEntry struct {
	name  string
	isDir bool
}

func (f fakeDirEntry) Name() string      { return f.name }
func (f fakeDirEntry) IsDir() bool       { return f.isDir }
func (f fakeDirEntry) Type() os.FileMode { return 0 }

func (f fakeDirEntry) Info() (os.FileInfo, error) { return nil, nil } //nolint:nilnil // os.DirEntry.Info contract allows nil,nil when not implemented.
