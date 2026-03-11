package state

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/utils"
)

// newMemoryStore creates a SQLiteBlockStore backed by an in-memory SQLite database.
func newMemoryStore(t *testing.T) *SQLiteBlockStore {
	t.Helper()

	fs := &utils.MoqFileSystem{}

	store, err := NewSQLiteBlockStore(fs, ":memory:")
	require.NoError(t, err)

	t.Cleanup(func() { _ = store.Close() })

	return store
}

// sampleRecord returns a populated Record for use in tests.
func sampleRecord(path string) Record {
	return Record{
		Path:         path,
		Subscription: "acme-platform",
		Source:       "skills/go-coding/system-prompt.md",
		BlockType:    "skills",
		Framework:    "claude-code",
		Checksum:     "abc123",
	}
}

// --- DefaultPath tests ---

func TestDefaultPath_ResolvingPathShouldUseXDGDataHomeWhenSet(t *testing.T) {
	envMgr := &moqEnv{values: map[string]string{"XDG_DATA_HOME": "/custom/data"}}
	userMgr := &moqUser{}

	path, err := DefaultPath(envMgr, userMgr)

	require.NoError(t, err)
	require.Equal(t, "/custom/data/cova/state.db", path)
}

func TestDefaultPath_ResolvingPathShouldFallBackToHomeDirWhenXDGIsEmpty(t *testing.T) {
	envMgr := &moqEnv{}
	userMgr := &moqUser{homeDir: "/home/testuser"}

	path, err := DefaultPath(envMgr, userMgr)

	require.NoError(t, err)
	require.Equal(t, "/home/testuser/.local/share/cova/state.db", path)
}

func TestDefaultPath_ResolvingPathShouldReturnErrorWhenHomeDirFails(t *testing.T) {
	envMgr := &moqEnv{}
	userMgr := &moqUser{err: errHomeDirFailed}

	_, err := DefaultPath(envMgr, userMgr)

	require.Error(t, err)
	require.Contains(t, err.Error(), "resolving home directory")
}

// --- RecordBatch tests ---

func TestRecordBatch_RecordingBatchShouldPersistAllRecords(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	records := []Record{
		sampleRecord("/dest/skills/go-coding/system-prompt.md"),
		sampleRecord("/dest/rules/no-comments/rule.md"),
	}

	err := store.RecordBatch(ctx, records)

	require.NoError(t, err)

	got, err := store.QueryBySubscriptionFramework(ctx, "acme-platform", "claude-code")
	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestRecordBatch_RecordingBatchShouldUpsertExistingRecords(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	path := "/dest/skills/go-coding/system-prompt.md"

	initial := sampleRecord(path)
	require.NoError(t, store.RecordBatch(ctx, []Record{initial}))

	updated := initial
	updated.Checksum = "newchecksum"

	require.NoError(t, store.RecordBatch(ctx, []Record{updated}))

	got, err := store.QueryByPath(ctx, path)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "newchecksum", got.Checksum)
}

func TestRecordBatch_RecordingEmptyBatchShouldDoNothing(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	err := store.RecordBatch(ctx, nil)

	require.NoError(t, err)
}

// --- QueryByPath tests ---

func TestQueryByPath_QueryingExistingPathShouldReturnRecord(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	path := "/dest/skills/go-coding/system-prompt.md"
	rec := sampleRecord(path)

	require.NoError(t, store.RecordBatch(ctx, []Record{rec}))

	got, err := store.QueryByPath(ctx, path)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, rec, *got)
}

func TestQueryByPath_QueryingMissingPathShouldReturnErrNotFound(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	got, err := store.QueryByPath(ctx, "/nonexistent/path")

	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, got)
}

// --- QueryBySubscriptionFramework tests ---

func TestQueryBySubscriptionFramework_QueryingShouldReturnMatchingRecords(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	matching := []Record{
		sampleRecord("/dest/skills/go-coding/system-prompt.md"),
		sampleRecord("/dest/rules/no-comments/rule.md"),
	}

	other := Record{
		Path:         "/dest/skills/other/prompt.md",
		Subscription: "other-sub",
		Source:       "skills/other/prompt.md",
		BlockType:    "skills",
		Framework:    "openai",
		Checksum:     "",
	}

	require.NoError(t, store.RecordBatch(ctx, append(matching, other)))

	got, err := store.QueryBySubscriptionFramework(ctx, "acme-platform", "claude-code")

	require.NoError(t, err)
	require.Len(t, got, 2)
}

func TestQueryBySubscriptionFramework_QueryingShouldReturnEmptySliceWhenNoneMatch(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	got, err := store.QueryBySubscriptionFramework(ctx, "no-such-sub", "no-such-framework")

	require.NoError(t, err)
	require.Empty(t, got)
}

// --- DeleteByPaths tests ---

func TestDeleteByPaths_DeletingShouldRemoveSpecifiedRecords(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	paths := []string{
		"/dest/skills/go-coding/system-prompt.md",
		"/dest/rules/no-comments/rule.md",
	}
	records := []Record{sampleRecord(paths[0]), sampleRecord(paths[1])}

	require.NoError(t, store.RecordBatch(ctx, records))
	require.NoError(t, store.DeleteByPaths(ctx, []string{paths[0]}))

	got, err := store.QueryBySubscriptionFramework(ctx, "acme-platform", "claude-code")
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, paths[1], got[0].Path)
}

func TestDeleteByPaths_DeletingEmptySliceShouldDoNothing(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	rec := sampleRecord("/dest/skills/go-coding/system-prompt.md")
	require.NoError(t, store.RecordBatch(ctx, []Record{rec}))

	err := store.DeleteByPaths(ctx, nil)

	require.NoError(t, err)

	got, err := store.QueryBySubscriptionFramework(ctx, "acme-platform", "claude-code")
	require.NoError(t, err)
	require.Len(t, got, 1)
}

func TestDeleteByPaths_DeletingNonExistentPathsShouldNotError(t *testing.T) {
	store := newMemoryStore(t)
	ctx := t.Context()

	err := store.DeleteByPaths(ctx, []string{"/nonexistent/path"})

	require.NoError(t, err)
}

// --- minimal inline mocks for EnvironmentManager / UserManager ---

var errHomeDirFailed = errors.New("home dir unavailable")

type moqEnv struct {
	values map[string]string
}

func (m *moqEnv) Getenv(key string) string {
	return m.values[key]
}

type moqUser struct {
	err     error
	homeDir string
}

func (m *moqUser) GetHomeDir() (string, error) {
	return m.homeDir, m.err
}

func (m *moqUser) GetConfigDir() (string, error) {
	return "", nil
}

func (m *moqUser) GetCurrentUsername() (string, error) {
	return "", nil
}
