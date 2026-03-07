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
- CLI docs: `docs/cova/` (consuming.md, configuration.md, workspaces.md, index.md, adapters.md, state.md, contributing.md)

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
- Config: `$XDG_CONFIG_HOME/cova/config.yaml`, has subscriptions + frameworks sections
- Workspace: `$XDG_CACHE_HOME/cova/repos/`, keyed by repo URL
- Manifest: `manifest.yaml` at repo root, `org` + `covens` (string or list)
- Naming segments: lowercase alphanumeric + hyphens, no leading/trailing/consecutive hyphens
