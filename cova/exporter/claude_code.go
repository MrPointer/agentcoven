package exporter

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// supportedClaudeCodeTypes maps block type names to their target subdirectory
// under ~/.claude/.
var supportedClaudeCodeTypes = map[string]string{
	"skills": "skills",
	"agents": "agents",
}

// claudeCodeExporter is the built-in exporter for the Claude Code agent.
type claudeCodeExporter struct {
	fs      utils.FileSystem
	homeDir string
}

var _ exporter = (*claudeCodeExporter)(nil)

// newClaudeCodeExporter creates a Claude Code exporter.
// homeDir is the user's home directory, used to build absolute target paths.
func newClaudeCodeExporter(fs utils.FileSystem, homeDir string) *claudeCodeExporter {
	return &claudeCodeExporter{fs: fs, homeDir: homeDir}
}

// apply computes file placements for all blocks in the request.
// Unsupported block types produce a per-block error result and do not abort processing.
func (a *claudeCodeExporter) apply(_ context.Context, req *ApplyRequest) (*ApplyResponse, error) {
	var results []BlockResult

	for blockType, blocks := range req.Blocks {
		subDir, supported := supportedClaudeCodeTypes[blockType]

		for _, b := range blocks {
			if !supported {
				errMsg := fmt.Sprintf("block type %q is not supported by the claude-code exporter", blockType)
				results = append(results, BlockResult{
					Name:  b.Name,
					Error: &errMsg,
				})

				continue
			}

			blockResult, err := a.buildBlockResult(req.Workspace, subDir, b)
			if err != nil {
				return nil, fmt.Errorf("building placements for block %q: %w", b.Name, err)
			}

			results = append(results, blockResult)
		}
	}

	return &ApplyResponse{Results: results}, nil
}

// buildBlockResult lists the files inside the block's source directory and
// constructs a placement for each one.
func (a *claudeCodeExporter) buildBlockResult(workspace, subDir string, b RequestBlock) (BlockResult, error) {
	blockSourceAbs := filepath.Join(workspace, b.Source)

	entries, err := a.fs.ReadDirectory(blockSourceAbs)
	if err != nil {
		return BlockResult{}, fmt.Errorf("reading block directory %q: %w", blockSourceAbs, err)
	}

	placements := make([]Placement, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		targetPath := filepath.Join(a.homeDir, ".claude", subDir, b.Name, entry.Name())
		sourcePath := filepath.Join(b.Source, entry.Name())

		placements = append(placements, Placement{
			Path:   targetPath,
			Source: sourcePath,
		})
	}

	return BlockResult{
		Name:       b.Name,
		Placements: placements,
	}, nil
}
