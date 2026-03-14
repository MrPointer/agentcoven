package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite" // SQLite driver registration.

	"github.com/MrPointer/agentcoven/cova/utils"
)

const (
	// memoryPath is the special SQLite path for an in-memory database.
	memoryPath = ":memory:"

	// createTableSQL is the schema for the blocks table.
	createTableSQL = `CREATE TABLE IF NOT EXISTS blocks (
		path         TEXT PRIMARY KEY,
		subscription TEXT NOT NULL,
		source        TEXT NOT NULL,
		block_type    TEXT NOT NULL,
		framework     TEXT NOT NULL,
		checksum      TEXT NOT NULL DEFAULT ''
	)`

	// upsertSQL inserts or replaces a record.
	upsertSQL = `INSERT INTO blocks (path, subscription, source, block_type, framework, checksum)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			subscription = excluded.subscription,
			source        = excluded.source,
			block_type    = excluded.block_type,
			framework     = excluded.framework,
			checksum      = excluded.checksum`

	// queryByPathSQL selects a single record by path.
	queryByPathSQL = `SELECT path, subscription, source, block_type, framework, checksum
		FROM blocks WHERE path = ?`

	// queryBySubFrameworkSQL selects all records for a subscription+framework.
	queryBySubFrameworkSQL = `SELECT path, subscription, source, block_type, framework, checksum
		FROM blocks WHERE subscription = ? AND framework = ?`
)

// SQLiteBlockStore is a BlockStore backed by a SQLite database.
type SQLiteBlockStore struct {
	db *sql.DB
}

var _ BlockStore = (*SQLiteBlockStore)(nil)

// NewSQLiteBlockStore opens (or creates) the SQLite database at path, creates parent
// directories as needed (skipped for :memory:), and initialises the schema.
func NewSQLiteBlockStore(fs utils.FileSystem, path string) (*SQLiteBlockStore, error) {
	if path != memoryPath {
		dir := filepath.Dir(path)
		if err := fs.CreateDirectory(dir); err != nil {
			return nil, fmt.Errorf("creating state directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening state database: %w", err)
	}

	if err := initSchema(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLiteBlockStore{db: db}, nil
}

// initSchema runs the CREATE TABLE IF NOT EXISTS statement.
func initSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("initialising state schema: %w", err)
	}

	return nil
}

// RecordBatch upserts all records in a single transaction.
func (s *SQLiteBlockStore) RecordBatch(ctx context.Context, records []Record) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, upsertSQL)
	if err != nil {
		return fmt.Errorf("preparing upsert statement: %w", errors.Join(err, tx.Rollback()))
	}

	defer stmt.Close()

	for _, r := range records {
		if _, err := stmt.ExecContext(
			ctx,
			r.Path,
			r.Subscription,
			r.Source,
			r.BlockType,
			r.Framework,
			r.Checksum,
		); err != nil {
			return fmt.Errorf("upserting record %q: %w", r.Path, errors.Join(err, tx.Rollback()))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing batch: %w", err)
	}

	return nil
}

// QueryByPath returns the record for the given absolute path.
// Returns ErrNotFound if no record exists for the path.
func (s *SQLiteBlockStore) QueryByPath(ctx context.Context, path string) (*Record, error) {
	row := s.db.QueryRowContext(ctx, queryByPathSQL, path)

	var r Record
	if err := row.Scan(&r.Path, &r.Subscription, &r.Source, &r.BlockType, &r.Framework, &r.Checksum); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("querying record by path %q: %w", path, err)
	}

	return &r, nil
}

// QueryBySubscriptionFramework returns all records for the given subscription and framework.
func (s *SQLiteBlockStore) QueryBySubscriptionFramework(
	ctx context.Context,
	subscription, framework string,
) ([]Record, error) {
	rows, err := s.db.QueryContext(ctx, queryBySubFrameworkSQL, subscription, framework)
	if err != nil {
		return nil, fmt.Errorf("querying records for subscription %q framework %q: %w", subscription, framework, err)
	}

	defer rows.Close()

	var records []Record

	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Path, &r.Subscription, &r.Source, &r.BlockType, &r.Framework, &r.Checksum); err != nil {
			return nil, fmt.Errorf("scanning record: %w", err)
		}

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating records: %w", err)
	}

	return records, nil
}

// DeleteByPaths removes all records with the given absolute paths.
func (s *SQLiteBlockStore) DeleteByPaths(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	placeholders := make([]string, len(paths))
	args := make([]any, len(paths))

	for i, p := range paths {
		placeholders[i] = "?"
		args[i] = p
	}

	query := fmt.Sprintf( //nolint:gosec // G201: placeholders are "?" literals, not user input.
		"DELETE FROM blocks WHERE path IN (%s)",
		strings.Join(placeholders, ", "),
	)

	if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("deleting records: %w", err)
	}

	return nil
}

// Close releases the underlying database connection.
func (s *SQLiteBlockStore) Close() error {
	return s.db.Close()
}
