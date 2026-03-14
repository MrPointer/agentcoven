// Package state manages the cova state database, tracking every file placed on disk.
package state

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

// ErrNotFound is returned by QueryByPath when no record exists for the given path.
var ErrNotFound = errors.New("record not found")

const (
	// dataDirName is the directory name under the XDG data root.
	dataDirName = "cova"
	// stateFileName is the state database file name.
	stateFileName = "state.db"
)

// Record represents a single row in the blocks table.
type Record struct {
	// Path is the absolute path where the block file was written.
	Path string
	// Subscription is the name of the subscription that owns this block.
	Subscription string
	// Source is the path of the block file within the coven repository.
	Source string
	// BlockType is the block type (e.g., skills, rules, agents).
	BlockType string
	// Agent is the target agent the block was applied to.
	Agent string
	// Checksum is the SHA-256 hash of the applied file contents.
	Checksum string
}

// BlockStore tracks the files cova places on disk.
type BlockStore interface {
	// RecordBatch upserts multiple records in a single transaction.
	RecordBatch(ctx context.Context, records []Record) error

	// QueryByPath returns the record for the given absolute path.
	// Returns ErrNotFound if no record exists for the path.
	QueryByPath(ctx context.Context, path string) (*Record, error)

	// QueryBySubscriptionAgent returns all records for the given subscription and agent.
	QueryBySubscriptionAgent(ctx context.Context, subscription, agent string) ([]Record, error)

	// DeleteByPaths removes all records with the given absolute paths.
	DeleteByPaths(ctx context.Context, paths []string) error

	// Close releases the database connection.
	Close() error
}

// DefaultPath resolves the state database path using XDG conventions.
// It checks $XDG_DATA_HOME first; if unset or empty, falls back to ~/.local/share.
func DefaultPath(
	envMgr osmanager.EnvironmentManager,
	userMgr osmanager.UserManager,
) (string, error) {
	dataHome := envMgr.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := userMgr.GetHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}

		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(dataHome, dataDirName, stateFileName), nil
}
