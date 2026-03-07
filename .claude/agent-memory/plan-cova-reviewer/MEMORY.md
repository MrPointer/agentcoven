# Cova Reviewer Memory

## Project Structure
- Go module: `github.com/MrPointer/agentcoven/cova` in `cova/` directory
- Go version: 1.25 (from go.mod)
- Dependencies: cobra, lipgloss, testify, yaml.v3 (indirect)
- Mockery config: `cova/.mockery.yml` — matryer template, `Moq` prefix, recursive, `all: true` for root package

## Key Interfaces
- `utils.FileSystem` in `cova/utils/filesystem.go` — large interface (13 methods) covering file/dir CRUD, temp files, read/write, path checks
  - `CreateTemporaryFile(dir, pattern string) (string, error)` — dir="" uses system temp
  - `WriteFile(path string, reader io.Reader) (int64, error)` — takes io.Reader, NOT []byte
  - `Rename(oldPath, newPath string) error`
  - `DefaultFileSystem` constructor takes `logger.Logger`
- `utils.Commander` in `cova/utils/commander.go` (mock exists at `cova/utils/commander_mock.go`)
- `osmanager.UserManager` — `GetHomeDir()`, `GetConfigDir()`, `GetCurrentUsername()`
  - `GetConfigDir()` wraps `os.UserConfigDir()` (OS-level, NOT XDG on macOS)
- `osmanager.EnvironmentManager` — `Getenv(key string) string`
- Mock files: `cova/utils/FileSystem_mock.go`, `cova/utils/commander_mock.go`

## Packages Not Yet Created (cova-add-v2)
- `cova/config/` — config package (sub-plan 03)
- `cova/lib/` — no lib packages exist yet
- `cova/utils/locker.go` — Locker interface (sub-plan 01, prerequisite for 03)

## Specs
- Repo spec: `docs/spec.md` — manifest structure, naming convention, block types, variants
- Client spec: `docs/client-spec.md` — subscriptions, application, adapters, conflict detection
- Cova docs: `docs/cova/` — index, consuming, contributing, configuration, workspaces, state, adapters

## Naming Convention (from spec.md)
- Pattern: `{org}-{coven}-{block-name}`
- Segments: lowercase alphanumeric + hyphens, no leading/trailing hyphens, no consecutive hyphens
- Subscription name: `{org}-{coven}` (from client-spec.md)

## Manifest (from spec.md)
- `manifest.yaml` at repo root
- Fields: `org` (required, naming segment), `covens` (required, string or list)
- String = single-coven, list = multi-coven
- Multi-coven: each name must have directory under `covens/`

## Code Conventions (from writing-go-code skill)
- DI via constructors, all external deps behind interfaces
- `Default*` naming for concrete types
- `var _ Interface = (*Concrete)(nil)` after struct
- Stateless packages can accept deps as function params (no struct needed)
- `samber/mo` for optionals
- 120 char line limit

## CLI Conventions (from developing-cli-apps skill)
- Cobra commands in `cmd/`, one file per command
- `Run` not `RunE`, handle errors inline
- Flags: kebab-case, bind to Viper

## Known Doc Issues
- `docs/cova/workspaces.md` references `[monorepo]: ../spec.md#monorepo` but no `#monorepo` anchor exists in spec.md (broken link)
- Workspaces doc explicitly says apply reads from refs "without requiring a checkout" — checkout-based workspace APIs will need rework for `apply`

## XDG Path Resolution
- Always needs both `EnvironmentManager` (for `$XDG_*` env vars) and `UserManager` (for home dir fallback)
- Config: `$XDG_CONFIG_HOME/cova/config.yaml` (default `~/.config/cova/config.yaml`)
- Cache: `$XDG_CACHE_HOME/cova/repos/` (default `~/.cache/cova/repos/`)
- Config repo URLs stored without protocol prefix (e.g., `github.com/acme/coven-blocks`)

## URL Normalization
- Must handle: https://, http://, file:// prefixes; .git suffix; trailing slashes; host case-insensitivity

## Skills Inventory
- Existing: writing-go-code, applying-effective-go, developing-cli-apps, writing-go-tests
- NOT existing: documenting-components (referenced in 07-documentation sub-plan but absent from `.claude/skills/`)
- NOT existing: testing-go-code (referenced in sub-plan 06 but absent from `.claude/skills/`)

## Review Patterns
- Sub-plans may deviate from docs (e.g., consuming.md says interactive prompt for multi-coven no-args, but sub-plan 05 chose error instead) — always cross-check
- Check skill references in sub-plans against actual `.claude/skills/` contents
- Test conventions: unit tests use `testing.Short()` for opt-out per writing-go-tests skill; build tags are a different pattern
- Watch for `RunE` vs `Run` — CLI skill mandates `Run`, sub-plans sometimes use `RunE` terminology
- Master plan (00-master.md) explicitly scopes out: apply, adapters, state DB, frameworks, interactive UI
- Multi-coven without args: error (not prompt) is intentional per master plan scope, but contradicts consuming.md
