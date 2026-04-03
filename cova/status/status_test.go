package status

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MrPointer/agentcoven/cova/state"
	"github.com/MrPointer/agentcoven/cova/utils"
	"github.com/MrPointer/agentcoven/cova/utils/logger"
	"github.com/MrPointer/agentcoven/cova/utils/osmanager"
)

// configYAML is a standard config with two subscriptions and two agents.
const configYAML = `
subscriptions:
  - name: acme-platform
    repo: github.com/acme/coven-blocks
    path: covens/platform
    ref: main
  - name: acme-frontend
    repo: github.com/acme/coven-blocks
    path: covens/frontend
    ref: v2.1.0
agents:
  - claude-code
  - cursor
`

// configOneSubNoRefNoPath is a config with one subscription without path or ref.
const configOneSubNoRefNoPath = `
subscriptions:
  - name: acme-platform
    repo: github.com/acme/coven-blocks
agents:
  - claude-code
`

// configNoAgents is a config with subscriptions but no agents.
const configNoAgents = `
subscriptions:
  - name: acme-platform
    repo: github.com/acme/coven-blocks
`

// hasSuffix reports whether s ends with suffix.
func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// makeEnvUserMgr returns standard env/user manager mocks returning no XDG overrides
// and home dir /home/user.
func makeEnvUserMgr() (osmanager.EnvironmentManager, osmanager.UserManager) {
	envMgr := &osmanager.MoqEnvironmentManager{
		GetenvFunc: func(key string) string { return "" },
	}
	userMgr := &osmanager.MoqUserManager{
		GetHomeDirFunc: func() (string, error) { return "/home/user", nil },
	}

	return envMgr, userMgr
}

// makeFS returns a filesystem mock that serves the given configYAML for config reads.
func makeFS(cfgYAML string) *utils.MoqFileSystem {
	return &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			if hasSuffix(path, "config.yaml") {
				return []byte(cfgYAML), nil
			}

			return nil, errors.New("unexpected path: " + path)
		},
	}
}

// makeDeps returns a Deps with the given mocks and a bytes.Buffer as Out.
func makeDeps(
	fs utils.FileSystem,
	store state.BlockStore,
	envMgr osmanager.EnvironmentManager,
	userMgr osmanager.UserManager,
	out *bytes.Buffer,
) Deps {
	return Deps{
		Logger:      logger.NoopLogger{},
		FileSystem:  fs,
		BlockStore:  store,
		EnvManager:  envMgr,
		UserManager: userMgr,
		Out:         out,
	}
}

// makeRecordsStore returns a BlockStore that returns the given records for any subscription query.
func makeRecordsStore(recordsBySubscription map[string][]state.Record) *state.MoqBlockStore {
	return &state.MoqBlockStore{
		QueryBySubscriptionFunc: func(ctx context.Context, subscription string) ([]state.Record, error) {
			return recordsBySubscription[subscription], nil
		},
	}
}

func TestRun_StatusShouldPrintNoSubscriptionsWhenConfigIsEmpty(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS("")
	out := &bytes.Buffer{}
	deps := makeDeps(fs, nil, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	require.Contains(t, out.String(), "No subscriptions")
}

func TestRun_StatusShouldPrintSubscriptionsSectionInDefaultMode(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}
	store := makeRecordsStore(nil)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "Subscriptions:")
	require.Contains(t, output, "acme-platform")
	require.Contains(t, output, "acme-frontend")
	require.Contains(t, output, "github.com/acme/coven-blocks")
}

func TestRun_StatusShouldIncludePathAndRefWhenSet(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}
	store := makeRecordsStore(nil)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "covens/platform")
	require.Contains(t, output, "@ main")
	require.Contains(t, output, "covens/frontend")
	require.Contains(t, output, "@ v2.1.0")
}

func TestRun_StatusShouldOmitPathAndRefWhenNotSet(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configOneSubNoRefNoPath)
	out := &bytes.Buffer{}
	store := makeRecordsStore(nil)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)

	output := out.String()
	require.NotContains(t, output, "@ ")
}

func TestRun_StatusShouldShowZeroBlocksWhenNoStateExists(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}
	deps := makeDeps(fs, nil, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	require.Contains(t, out.String(), "Applied: 0 blocks")
}

func TestRun_StatusShouldShowZeroBlocksWhenBlockStoreIsNil(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configOneSubNoRefNoPath)
	out := &bytes.Buffer{}
	deps := makeDeps(fs, nil, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	require.Contains(t, out.String(), "Applied: 0 blocks")
}

func TestRun_StatusShouldPrintAppliedCountWithTypeBreakdown(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}

	records := map[string][]state.Record{
		"acme-platform": {
			{
				Subscription: "acme-platform",
				Source:       "skills/acme-platform-code-review/SKILL.md",
				BlockType:    "skills",
				Agent:        "claude-code",
			},
			{
				Subscription: "acme-platform",
				Source:       "skills/acme-platform-testing/SKILL.md",
				BlockType:    "skills",
				Agent:        "claude-code",
			},
			{
				Subscription: "acme-platform",
				Source:       "rules/acme-platform-go-conventions/RULES.md",
				BlockType:    "rules",
				Agent:        "claude-code",
			},
		},
		"acme-frontend": {
			{
				Subscription: "acme-frontend",
				Source:       "agents/acme-frontend-designer/AGENT.md",
				BlockType:    "agents",
				Agent:        "claude-code",
			},
		},
	}

	store := makeRecordsStore(records)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "Applied: 4 blocks")
	require.Contains(t, output, "1 agents")
	require.Contains(t, output, "1 rules")
	require.Contains(t, output, "2 skills")
}

func TestRun_StatusShouldDeduplicateBlocksWithMultipleFiles(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configOneSubNoRefNoPath)
	out := &bytes.Buffer{}

	// Two records for the same block (multiple files placed by one block).
	records := map[string][]state.Record{
		"acme-platform": {
			{
				Subscription: "acme-platform",
				Source:       "skills/my-skill/SKILL.md",
				BlockType:    "skills",
				Agent:        "claude-code",
			},
			{
				Subscription: "acme-platform",
				Source:       "skills/my-skill/extra.md",
				BlockType:    "skills",
				Agent:        "claude-code",
			},
		},
	}

	store := makeRecordsStore(records)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	require.Contains(t, out.String(), "Applied: 1 blocks")
}

func TestRun_StatusShouldPrintVerboseBlocksGroupedBySubscriptionAndType(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}

	records := map[string][]state.Record{
		"acme-platform": {
			{
				Subscription: "acme-platform",
				Source:       "skills/acme-platform-code-review/SKILL.md",
				BlockType:    "skills",
				Agent:        "claude-code",
			},
			{
				Subscription: "acme-platform",
				Source:       "rules/acme-platform-go-conventions/RULES.md",
				BlockType:    "rules",
				Agent:        "claude-code",
			},
		},
		"acme-frontend": {
			{
				Subscription: "acme-frontend",
				Source:       "agents/acme-frontend-designer/AGENT.md",
				BlockType:    "agents",
				Agent:        "claude-code",
			},
		},
	}

	store := makeRecordsStore(records)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, true)

	require.NoError(t, err)

	output := out.String()

	require.Contains(t, output, "acme-platform (2 blocks):")
	require.Contains(t, output, "  skills:")
	require.Contains(t, output, "    acme-platform-code-review")
	require.Contains(t, output, "  rules:")
	require.Contains(t, output, "    acme-platform-go-conventions")
	require.Contains(t, output, "acme-frontend (1 blocks):")
	require.Contains(t, output, "  agents:")
	require.Contains(t, output, "    acme-frontend-designer")
}

func TestRun_StatusShouldShowNoBlocksAppliedInVerboseModeWhenSubscriptionHasNone(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}
	store := makeRecordsStore(nil)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, true)

	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "acme-platform (0 blocks):")
	require.Contains(t, output, "  No blocks applied")
	require.Contains(t, output, "acme-frontend (0 blocks):")
}

func TestRun_StatusShouldPrintAgentsLine(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}
	store := makeRecordsStore(nil)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)

	output := out.String()
	require.Contains(t, output, "Agents: claude-code, cursor")
}

func TestRun_StatusShouldOmitAgentsLineWhenNoneConfigured(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configNoAgents)
	out := &bytes.Buffer{}
	store := makeRecordsStore(nil)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	require.NotContains(t, out.String(), "Agents:")
}

func TestRun_StatusShouldErrorWhenConfigLoadFails(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()

	fs := &utils.MoqFileSystem{
		PathExistsFunc: func(path string) (bool, error) { return true, nil },
		ReadFileContentsFunc: func(path string) ([]byte, error) {
			return nil, errors.New("disk read error")
		},
	}

	out := &bytes.Buffer{}
	deps := makeDeps(fs, nil, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.Error(t, err)
	require.Contains(t, err.Error(), "loading config")
}

func TestRun_StatusShouldSkipMalformedSourceAndWarn(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configOneSubNoRefNoPath)
	out := &bytes.Buffer{}

	var warnings []string

	log := &logger.MoqLogger{
		WarningFunc: func(format string, args ...any) { warnings = append(warnings, format) },
		CloseFunc:   func() error { return nil },
	}

	// One malformed source (only 1 component) and one valid one.
	records := map[string][]state.Record{
		"acme-platform": {
			{Subscription: "acme-platform", Source: "malformed", BlockType: "skills", Agent: "claude-code"},
			{
				Subscription: "acme-platform",
				Source:       "skills/valid-block/SKILL.md",
				BlockType:    "skills",
				Agent:        "claude-code",
			},
		},
	}

	store := makeRecordsStore(records)
	deps := Deps{
		Logger:      log,
		FileSystem:  fs,
		BlockStore:  store,
		EnvManager:  envMgr,
		UserManager: userMgr,
		Out:         out,
	}

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	require.NotEmpty(t, warnings, "expected a warning for the malformed source")
	require.Contains(t, out.String(), "Applied: 1 blocks")
}

func TestRun_StatusShouldWarnAndContinueWhenStateQueryFails(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configOneSubNoRefNoPath)
	out := &bytes.Buffer{}

	var warnings []string

	log := &logger.MoqLogger{
		WarningFunc: func(format string, args ...any) { warnings = append(warnings, format) },
		CloseFunc:   func() error { return nil },
	}

	store := &state.MoqBlockStore{
		QueryBySubscriptionFunc: func(ctx context.Context, subscription string) ([]state.Record, error) {
			return nil, errors.New("db connection lost")
		},
	}

	deps := Deps{
		Logger:      log,
		FileSystem:  fs,
		BlockStore:  store,
		EnvManager:  envMgr,
		UserManager: userMgr,
		Out:         out,
	}

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	require.NotEmpty(t, warnings, "expected a warning for the query failure")
	require.Contains(t, out.String(), "Applied: 0 blocks")
}

func TestRun_StatusShouldNotPrintVerboseSectionInDefaultMode(t *testing.T) {
	ctx := t.Context()
	envMgr, userMgr := makeEnvUserMgr()
	fs := makeFS(configYAML)
	out := &bytes.Buffer{}

	records := map[string][]state.Record{
		"acme-platform": {
			{
				Subscription: "acme-platform",
				Source:       "skills/acme-platform-code-review/SKILL.md",
				BlockType:    "skills",
				Agent:        "claude-code",
			},
		},
	}

	store := makeRecordsStore(records)
	deps := makeDeps(fs, store, envMgr, userMgr, out)

	err := Run(ctx, deps, false)

	require.NoError(t, err)
	// In default mode, per-subscription block breakdowns must not appear.
	require.NotContains(t, out.String(), "acme-platform-code-review")
}
