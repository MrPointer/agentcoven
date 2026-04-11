---
name: Codebase Risk Patterns
description: Known complexity hotspots and risk patterns in the agentcoven/cova codebase for risk reviews
type: project
---

## Interface Mock Blast Radius

Mockery (matryer template, `Moq` prefix) generates mocks co-located with interfaces. When a public interface gains
methods, the regenerated mock changes its struct fields. Existing test files that instantiate the mock via struct
literal will still compile (new fields default to nil), but calling the unstubbed method panics at runtime.

Key interfaces and their consumer counts (as of 2026-04-04):
- `exporter.Dispatcher`: 16 instantiations across `add/`, `apply/`, `remove/` test files
- `osmanager.ProgramQuery`: used in multiple test files across the codebase
- `utils.FileSystem`, `utils.Locker`, `utils.Commander`: broadly used

**How to apply:** When reviewing plans that modify interfaces, always check consumer count and verify the plan accounts
for mock regeneration AND cross-package test health.

## Exporter Package Structure

- `exporter/` package has internal `exporter` interface (lowercase, unexported) and public `Dispatcher` interface
- Built-in exporters registered in `DefaultDispatcher.builtins` map (currently only `claude-code`)
- External exporters resolved via `cova-exporter-{name}` on PATH using `ProgramQuery.GetProgramPath`
- `ProgramQuery` currently only does exact-name lookup (`exec.LookPath`), no prefix/glob scanning
- Adding PATH scanning (e.g., `FindProgramsByPrefix`) is qualitatively different from existing capabilities

## Config Package Pattern

- Locked read-modify-write via `locker.WithLock`
- `UpsertSubscription` / `RemoveSubscription` are the established patterns
- Atomic writes via temp file + rename
- Adding `AddAgents` / `RemoveAgents` is straightforward pattern-following

## Orchestration Package Convention

- Each top-level command has a matching package: `add/`, `remove/`, `apply/`, `status/`
- Each exposes a `Deps` struct and a `Run` function
- For the exporter management commands, orchestration lives in the existing `exporter/` package (not a separate
  package) since `exporter/` doesn't import `config/`, so no circular dependency. Functions: `exporter.Add`,
  `exporter.Remove`, `exporter.List` with a shared `exporter.Deps` struct.
- This breaks the one-package-per-command pattern but avoids creating a new package with a confusing name.
  The `exporter/` package now serves dual roles: protocol types/routing AND command orchestration.
