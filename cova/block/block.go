// Package block provides types and functions for discovering and resolving blocks in a coven directory.
package block

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// Block represents a single block discovered within a coven directory.
type Block struct {
	// Name is the block's namespaced name, taken directly from the block directory name.
	Name string

	// Type is the block's type directory name (e.g., "skills", "rules", "agents").
	Type string

	// SourceDir is the path to the block directory relative to the coven root
	// (e.g., "skills/acme-platform-code-review").
	SourceDir string
}

// Discover scans a coven root directory and returns all blocks grouped by type.
//
// It skips dot-prefixed directories (e.g., .git) and non-directory entries at
// the coven root level. Within each type directory, non-directory entries are
// also ignored. An empty map is returned when no blocks are found.
func Discover(fs utils.FileSystem, covenRoot string) (map[string][]Block, error) {
	topEntries, err := fs.ReadDirectory(covenRoot)
	if err != nil {
		return nil, fmt.Errorf("reading coven root %q: %w", covenRoot, err)
	}

	result := make(map[string][]Block)

	for _, entry := range topEntries {
		if !entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		typeDir := entry.Name()
		typePath := filepath.Join(covenRoot, typeDir)

		blockEntries, err := fs.ReadDirectory(typePath)
		if err != nil {
			return nil, fmt.Errorf("reading type directory %q: %w", typePath, err)
		}

		for _, blockEntry := range blockEntries {
			if !blockEntry.IsDir() {
				continue
			}

			b := Block{
				Name:      blockEntry.Name(),
				Type:      typeDir,
				SourceDir: filepath.Join(typeDir, blockEntry.Name()),
			}

			result[typeDir] = append(result[typeDir], b)
		}
	}

	return result, nil
}
