# Plan Architect Reviewer Memory

## Project: AgentCoven (cova CLI)

### Key File Locations
- Go module root: `cova/` (module: `github.com/MrPointer/agentcoven/cova`)
- Utils interfaces: `cova/utils/filesystem.go` (FileSystem), `cova/utils/commander.go` (Commander)
- OS manager interfaces: `cova/utils/osmanager/osmanager.go` (UserManager, EnvironmentManager, ProgramQuery, OsManager)
- Logger: `cova/utils/logger/logger.go` (Logger interface + NoopLogger)
- Root command: `cova/cmd/root.go` (package-level var `rootCmd`, no DI)
- Mockery config: `cova/.mockery.yml` (matryer template, MoqPrefix, recursive, explicit package entries)
- Specs: `docs/spec.md` (repo spec), `docs/client-spec.md` (client spec)
- CLI docs: `docs/cova/` (consuming.md, configuration.md, workspaces.md, index.md, exporters.md, state.md, contributing.md)

### Architecture Patterns
- Interfaces in `utils/` package, `Default*` concrete types, `NewDefault*(logger)` constructors
- Mocks: matryer template, `Moq` prefix, files named `{InterfaceName}_mock.go`
- `osmanager` sub-interfaces: UserManager, EnvironmentManager, ProgramQuery compose into OsManager
- DefaultOsManager takes logger, commander, fileSystem in constructor
- Root Cobra command is a package-level var with no DI; subcommands must construct deps in RunE

### Mockery Config Caveat
- `.mockery.yml` has `all: true` + `recursive: true` at top level BUT also explicit package entries
- New packages may need explicit entries in `.mockery.yml` to get mocks generated
- Verified: `cova/` package entry has `all: true`, `cova/cli` has `all: false`

### Dependencies (go.mod)
- `github.com/spf13/cobra v1.10.2`
- `gopkg.in/yaml.v3 v3.0.1` (indirect, via cobra/testify)
- `github.com/stretchr/testify v1.11.1`
- `github.com/charmbracelet/lipgloss v1.1.0`
- Go version: 1.25

### Spec Key Points
- Subscription fields: name (req), repo (req), path (opt), ref (opt)
- Config: `$XDG_CONFIG_HOME/cova/config.yaml`, has subscriptions + agents sections
- Workspace: `$XDG_CACHE_HOME/cova/repos/`, keyed by repo URL
- Manifest: `manifest.yaml` at repo root, `org` + `covens` (string or list)
- Naming segments: lowercase alphanumeric + hyphens, no leading/trailing/consecutive hyphens

### cova-apply Plan Review Notes (2026-03-09)
- `workspace.Ensure` does clone/fetch + optional checkout; apply uses worktrees instead of checkout
- `manifest.Parse` at `cova/manifest/manifest.go` - pure function (FileSystem + repoRoot path)
- `add.Deps` pattern: struct with interface fields, `Run` function takes ctx + deps + args
- State DB: `$XDG_DATA_HOME/cova/state.db`, `modernc.org/sqlite` (pure Go, no CGo)
- Block discovery (sub-plan 04): stateless functions, not interfaces -- affects testability of apply orchestration
- Adapter protocol schema divergence: sub-plan 05 redefines `workspace` field as coven root, not workspace root

### Exporter Package Architecture (2026-04-11)
- `cova/exporter/` files: `exporter.go` (types + internal interface), `dispatcher.go` (Dispatcher interface + DefaultDispatcher), `claude_code.go`, `external.go`
- `exporter` package does NOT import `config`; `config` does NOT import `exporter` -- no circular dep risk
- `DefaultDispatcher` constructor: `NewDefaultDispatcher(programQuery, commander, fs, homeDir)`
- Orchestration convention: each command has own package (`add/`, `remove/`, `apply/`, `status/`) with `Deps` struct + `Run` function
- Exporter management plan (cova-exporter-management) puts orchestration in `exporter/` instead of a new package -- breaks convention but avoids naming collision
- `status` package `Deps` includes `Out io.Writer` for testable output -- precedent for list-style commands
- `ProgramQuery` interface in `osmanager.go` has: GetProgramPath, ProgramExists, GetProgramVersion -- no prefix/glob scanning yet
