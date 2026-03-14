package e2e_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/apply"
	"github.com/MrPointer/agentcoven/cova/config"
	"github.com/MrPointer/agentcoven/cova/exporter"
	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
	"github.com/MrPointer/agentcoven/cova/workspace"
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

// setupApplyEnv extends setupIsolatedEnv with XDG_DATA_HOME and HOME isolation required by apply.
// It also sets HOME so the Claude Code exporter writes into a temp dir rather than the real home.
// Returns the temp home directory path so callers can verify placed files.
func setupApplyEnv(t *testing.T) string {
	t.Helper()

	setupIsolatedEnv(t)

	tempHome := t.TempDir()
	dataHome := filepath.Join(t.TempDir(), "data")

	require.NoError(t, os.MkdirAll(filepath.Join(dataHome, "cova"), 0o755))

	t.Setenv("XDG_DATA_HOME", dataHome)
	t.Setenv("HOME", tempHome)

	return tempHome
}

// newApplyDeps builds an apply.Deps with real Default* dependencies.
// homeDir is passed to the dispatcher so the Claude Code exporter writes to the temp home.
func newApplyDeps(t *testing.T, homeDir string) apply.Deps {
	t.Helper()

	log := logger.NoopLogger{}
	fs := utils.NewDefaultFileSystem(log)
	cmdr := utils.NewDefaultCommander(log)
	osMgr := osmanager.NewDefaultOsManager(log, cmdr, fs)

	statePath, err := state.DefaultPath(osMgr, osMgr)
	require.NoError(t, err)

	blockStore, err := state.NewSQLiteBlockStore(fs, statePath)
	require.NoError(t, err)

	t.Cleanup(func() { _ = blockStore.Close() })

	dispatcher := exporter.NewDefaultDispatcher(osMgr, cmdr, fs, homeDir)

	return apply.Deps{
		Logger:      log,
		FileSystem:  fs,
		Locker:      utils.NewDefaultLocker(),
		Git:         workspace.NewDefaultGit(cmdr, fs),
		BlockStore:  blockStore,
		Dispatcher:  dispatcher,
		EnvManager:  osMgr,
		UserManager: osMgr,
	}
}

// writeConfig writes a config.Config to disk at the XDG_CONFIG_HOME path so apply can load it.
func writeConfig(t *testing.T, cfg config.Config) {
	t.Helper()

	log := logger.NoopLogger{}
	fs := utils.NewDefaultFileSystem(log)
	cmdr := utils.NewDefaultCommander(log)
	osMgr := osmanager.NewDefaultOsManager(log, cmdr, fs)

	cfgPath, err := config.DefaultPath(osMgr, osMgr)
	require.NoError(t, err)

	require.NoError(t, config.Save(fs, cfgPath, cfg))
}

// createCovenRepoWithSkills creates a single-coven git repo that contains a skills block.
// Returns the repo directory and the skill file content for assertion purposes.
func createCovenRepoWithSkills(t *testing.T, org, coven, skillName string) (repoDir, skillContent string) {
	t.Helper()

	skillContent = "# " + skillName + " skill content"

	repoDir = createGitRepo(t, map[string]string{
		"manifest.yaml":                     singleCovenManifest(org, coven),
		"skills/" + skillName + "/SKILL.md": skillContent,
	})

	return repoDir, skillContent
}

// createCovenRepoWithAgents creates a single-coven git repo that contains an agents block.
func createCovenRepoWithAgents(t *testing.T, org, coven, agentName string) string {
	t.Helper()

	return createGitRepo(t, map[string]string{
		"manifest.yaml":                     singleCovenManifest(org, coven),
		"agents/" + agentName + "/agent.md": "# " + agentName + " agent",
	})
}

// createCovenRepoWithMixedBlocks creates a single-coven git repo with skills, agents, and
// an unsupported block type (rules) to exercise the mixed-type path.
func createCovenRepoWithMixedBlocks(t *testing.T, org, coven string) string {
	t.Helper()

	return createGitRepo(t, map[string]string{
		"manifest.yaml":            singleCovenManifest(org, coven),
		"skills/my-skill/SKILL.md": "skill content",
		"agents/my-agent/agent.md": "agent content",
		"rules/my-rule/RULE.md":    "rule content",
	})
}

// claudeCodeSkillPath returns the expected target path for a skill file placed by the Claude Code exporter.
func claudeCodeSkillPath(homeDir, skillName, fileName string) string {
	return filepath.Join(homeDir, ".claude", "skills", skillName, fileName)
}

// claudeCodeAgentPath returns the expected target path for an agent file placed by the Claude Code exporter.
func claudeCodeAgentPath(homeDir, agentName, fileName string) string {
	return filepath.Join(homeDir, ".claude", "agents", agentName, fileName)
}

// resolveWorkspacePath returns the on-disk workspace path for a repo URL.
// This matches the path that workspace.Ensure and workspace.NormalizeURL produce.
func resolveWorkspacePath(t *testing.T, repoURL string) string {
	t.Helper()

	log := logger.NoopLogger{}
	fs := utils.NewDefaultFileSystem(log)
	cmdr := utils.NewDefaultCommander(log)
	osMgr := osmanager.NewDefaultOsManager(log, cmdr, fs)

	basePath, err := workspace.DefaultBasePath(osMgr, osMgr)
	require.NoError(t, err)

	normalized, err := workspace.NormalizeURL(repoURL)
	require.NoError(t, err)

	return filepath.Join(basePath, normalized)
}

// openBlockStore opens the state database at the XDG_DATA_HOME path for assertion queries.
func openBlockStore(t *testing.T) state.BlockStore {
	t.Helper()

	log := logger.NoopLogger{}
	fs := utils.NewDefaultFileSystem(log)
	cmdr := utils.NewDefaultCommander(log)
	osMgr := osmanager.NewDefaultOsManager(log, cmdr, fs)

	statePath, err := state.DefaultPath(osMgr, osMgr)
	require.NoError(t, err)

	store, err := state.NewSQLiteBlockStore(fs, statePath)
	require.NoError(t, err)

	t.Cleanup(func() { _ = store.Close() })

	return store
}
