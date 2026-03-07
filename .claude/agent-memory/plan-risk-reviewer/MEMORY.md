# Risk Reviewer Memory

## Project: AgentCoven / cova CLI

### Tech Stack
- Go 1.25, module at `github.com/MrPointer/agentcoven/cova` (in `cova/` directory)
- Cobra for CLI, yaml.v3 (indirect dep), lipgloss, testify
- Mockery: matryer template, `Moq` prefix, config at `cova/.mockery.yml`
- DI pattern: interface + `Default*` concrete type + `NewDefault*(logger)` constructor

### Codebase Structure
- `cova/cmd/root.go`: package-level `var rootCmd` -- NO DI wiring, bare `rootCmd.Execute()`
- `cova/utils/`: FileSystem (13 methods), Commander (1 method) -- both with Default* impls
- `cova/utils/osmanager/`: UserManager, EnvironmentManager, ProgramQuery, OsManager (composite)
- `cova/utils/logger/`: Logger interface, CliLogger, NoopLogger

### Known Risk Patterns
- **go.mod conflicts in parallel agent execution**: Multiple agents adding/promoting deps in parallel always causes merge conflicts. Always assign go.mod ownership to one agent.
- **Cross-device rename**: Atomic write via temp+rename fails if temp dir != target dir filesystem. Must create temp file in same directory as target.
- **Mockery auto-discovery**: `.mockery.yml` uses `recursive: true` on top-level cova package. New sub-packages should be auto-discovered, but verify mock generation works for each new interface.
- **Cobra DI wiring gap**: Root command has no DI infrastructure. Any new subcommand needs explicit design for how deps are constructed and injected.
- **file:// vs https:// URL normalization**: E2E tests use file:// URLs but production uses https://. These normalize very differently. Ensure unit tests cover the production case.

### Spec Notes
- Client spec subscription fields: name (req), repo (req), path (opt), ref (opt)
- Subscription name uniqueness is NOT explicitly scoped in the spec -- ambiguous whether name must be globally unique or name+repo unique
- `consuming.md` describes `add` as also applying blocks + interactive prompts -- both are future scope, not current implementation

See also: [cova-add-v2-review.md](cova-add-v2-review.md) for detailed first review findings.
