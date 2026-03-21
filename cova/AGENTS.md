# Agents Guide — cova

Reference CLI for the AgentCoven specification, written in Go.
Module: `github.com/MrPointer/agentcoven/cova`.

## Package Layout

```
cmd/              Cobra command definitions (root, add, apply, remove)
add/              Add command orchestration
apply/            Apply command orchestration
remove/           Remove command orchestration
block/            Block discovery and variant resolution
config/           YAML config management (subscriptions, agents)
exporter/         Block placement routing and exporter implementations
manifest/         Coven manifest parsing
state/            SQLite-backed block state tracking
workspace/        Git repository cloning and worktree management
utils/            Shared utilities
  logger/         Leveled CLI logger with lipgloss styling
  osmanager/      OS queries (env, user, program lookup)
e2e/              End-to-end tests
```

## Design Principles

All core logic lives in public packages — not `internal/` — so that
third-party tools can import and reuse cova's functionality as a library.
The CLI commands in `cmd/` are thin wrappers: construct dependencies,
call the orchestration package, exit. If logic can't be reached without
the CLI, it's in the wrong place.

Each package owns a single concern and exposes interfaces for its
contract. Packages depend on each other's interfaces, never on
concrete types from other packages.

## Components

### Command Flow

Cobra commands in `cmd/` construct a `Deps` struct and delegate to
the corresponding orchestration package (`add.Run`, `apply.Run`,
`remove.Run`).

### Exporter Routing

`exporter.Dispatcher` resolves an agent name to an exporter:

1. Check built-in map (currently: `"claude-code"`)
2. If not built-in, look for `cova-exporter-{agent}` on `$PATH`
3. Invoke via the exporter protocol (JSON over stdin/stdout)

The Claude Code exporter supports `skills` and `agents` block types.
Not all agents support all block types — unsupported types get
a per-block error, not a fatal failure.

### State

SQLite database at `$XDG_DATA_HOME/cova/state.db`.
The `BlockStore` interface tracks which files were placed, by whom,
and with what checksum — used for conflict detection during apply.

### Config

YAML at `$XDG_CONFIG_HOME/cova/config.yaml`.
Stores subscriptions and target agent list.
Writes are atomic (temp file + rename) under a file lock.
