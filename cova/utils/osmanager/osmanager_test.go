package osmanager

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
)

func TestDefaultOsManager_FindingProgramsByPrefixShouldReturnMatchingExecutables(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()

	createExecutable(t, filepath.Join(binDir, "cova-exporter-foo"))
	createExecutable(t, filepath.Join(binDir, "cova-exporter-bar"))
	createFile(t, filepath.Join(binDir, "other-tool"))

	t.Setenv("PATH", binDir)

	mgr := newOsManager(t)

	results, err := mgr.FindProgramsByPrefix("cova-exporter-")

	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Contains(t, results, filepath.Join(binDir, "cova-exporter-foo"))
	require.Contains(t, results, filepath.Join(binDir, "cova-exporter-bar"))
}

func TestDefaultOsManager_FindingProgramsByPrefixShouldDeduplicateByBaseName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	createExecutable(t, filepath.Join(dir1, "cova-exporter-foo"))
	createExecutable(t, filepath.Join(dir2, "cova-exporter-foo"))

	t.Setenv("PATH", dir1+string(filepath.ListSeparator)+dir2)

	mgr := newOsManager(t)

	results, err := mgr.FindProgramsByPrefix("cova-exporter-")

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, filepath.Join(dir1, "cova-exporter-foo"), results[0])
}

func TestDefaultOsManager_FindingProgramsByPrefixShouldSkipNonExecutableFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if runtime.GOOS == "windows" {
		t.Skip("executable bit checks are not applicable on Windows")
	}

	binDir := t.TempDir()

	createExecutable(t, filepath.Join(binDir, "cova-exporter-foo"))
	createFile(t, filepath.Join(binDir, "cova-exporter-bar"))

	t.Setenv("PATH", binDir)

	mgr := newOsManager(t)

	results, err := mgr.FindProgramsByPrefix("cova-exporter-")

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, filepath.Join(binDir, "cova-exporter-foo"), results[0])
}

func TestDefaultOsManager_FindingProgramsByPrefixShouldSkipUnreadablePathDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()

	createExecutable(t, filepath.Join(binDir, "cova-exporter-foo"))

	t.Setenv("PATH", "/nonexistent-path-dir-xyz"+string(filepath.ListSeparator)+binDir)

	mgr := newOsManager(t)

	results, err := mgr.FindProgramsByPrefix("cova-exporter-")

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, filepath.Join(binDir, "cova-exporter-foo"), results[0])
}

func TestDefaultOsManager_FindingProgramsByPrefixShouldReturnEmptyResultWhenPathIsEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Setenv("PATH", "")

	mgr := newOsManager(t)

	results, err := mgr.FindProgramsByPrefix("cova-exporter-")

	require.NoError(t, err)
	require.Empty(t, results)
}

func TestDefaultOsManager_FindingProgramsByPrefixShouldSkipDirectoryEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	binDir := t.TempDir()

	subDir := filepath.Join(binDir, "cova-exporter-subdir")
	require.NoError(t, os.Mkdir(subDir, 0o755))
	createExecutable(t, filepath.Join(binDir, "cova-exporter-foo"))

	t.Setenv("PATH", binDir)

	mgr := newOsManager(t)

	results, err := mgr.FindProgramsByPrefix("cova-exporter-")

	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, filepath.Join(binDir, "cova-exporter-foo"), results[0])
}

// newOsManager creates a DefaultOsManager backed by a real filesystem and a no-op logger.
func newOsManager(t *testing.T) *DefaultOsManager {
	t.Helper()

	noopLogger := &logger.MoqLogger{
		DebugFunc:   func(format string, args ...any) {},
		InfoFunc:    func(format string, args ...any) {},
		WarningFunc: func(format string, args ...any) {},
		ErrorFunc:   func(format string, args ...any) {},
		SuccessFunc: func(format string, args ...any) {},
		TraceFunc:   func(format string, args ...any) {},
		CloseFunc:   func() error { return nil },
	}

	return NewDefaultOsManager(noopLogger, nil, utils.NewDefaultFileSystem(noopLogger))
}

// createExecutable creates a file with execute permissions at the given path.
func createExecutable(t *testing.T, path string) {
	t.Helper()

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(path, 0o755))
}

// createFile creates a regular (non-executable) file at the given path.
func createFile(t *testing.T, path string) {
	t.Helper()

	f, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, os.Chmod(path, 0o644))
}
