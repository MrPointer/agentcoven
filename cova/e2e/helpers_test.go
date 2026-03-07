package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// singleCovenManifest returns a manifest.yaml body for a single-coven repo.
func singleCovenManifest(org, coven string) string {
	return "org: " + org + "\ncovens: " + coven + "\n"
}

// multiCovenManifest returns a manifest.yaml body for a multi-coven repo.
func multiCovenManifest(org string, covens []string) string {
	var b strings.Builder

	b.WriteString("org: " + org + "\ncovens:\n")

	for _, c := range covens {
		b.WriteString("  - " + c + "\n")
	}

	return b.String()
}

// createGitRepo initialises a git repo in a temp dir, writes the given files, and commits them.
// It returns the directory path. Files map keys are relative paths; values are file contents.
func createGitRepo(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()

	gitCmd(t, dir, "init")
	gitCmd(t, dir, "config", "user.email", "test@test.com")
	gitCmd(t, dir, "config", "user.name", "Test")
	gitCmd(t, dir, "config", "commit.gpgsign", "false")

	for relPath, content := range files {
		absPath := filepath.Join(dir, relPath)
		require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
		require.NoError(t, os.WriteFile(absPath, []byte(content), 0o644))
	}

	gitCmd(t, dir, "add", ".")
	gitCmd(t, dir, "commit", "-m", "init")

	return dir
}

// createSingleCovenRepo creates a git repo with a single-coven manifest.
func createSingleCovenRepo(t *testing.T, org, coven string) string {
	t.Helper()

	return createGitRepo(t, map[string]string{
		"manifest.yaml": singleCovenManifest(org, coven),
	})
}

// createMultiCovenRepo creates a git repo with a multi-coven manifest and coven directories.
func createMultiCovenRepo(t *testing.T, org string, covens []string) string {
	t.Helper()

	files := map[string]string{
		"manifest.yaml": multiCovenManifest(org, covens),
	}
	for _, c := range covens {
		files[filepath.Join("covens", c, ".gitkeep")] = ""
	}

	return createGitRepo(t, files)
}

// fileURL returns a file:// URL for the given directory.
func fileURL(dir string) string {
	return "file://" + dir
}

// gitCmd runs a git command in the given directory and fails the test on error.
func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v failed: %s", args, string(out))
}
