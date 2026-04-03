# Agents Guide

AgentCoven is an open specification for sharing AI agent building blocks
(skills, rules, agents) through git repositories.
The project has two parts: the specification and a reference CLI.

## Repository Layout

```
docs/             Specification and documentation
  spec.md         Repository specification — how covens are structured
  client-spec.md  Client specification — how tools consume covens
  cova/           Documentation for the reference CLI
schemas/          JSON schemas for the exporter protocol
cova/             Reference CLI implementation (Go)
```

## Specifications

There are two distinct specs:

- **Repository spec** (`docs/spec.md`) — defines coven repository structure,
  manifests, block types, naming conventions, and agent variants.
  The contract for coven maintainers.
- **Client spec** (`docs/client-spec.md`) — defines application semantics,
  local configuration, conflict detection, contributing, and the exporter
  protocol. The contract for tool authors.

When implementing CLI features, the client spec is normative.
If the code and the spec disagree, the spec wins — update the code.

## Key Concepts

- **Coven** — a self-contained collection of shared blocks.
  A git repository contains one or more covens
  (single-coven or multi-coven layout).
- **Block** — a skill, rule, agent definition, or custom type.
  Each block is a directory within its type directory.
- **Exporter** — a function that maps blocks to filesystem placements
  for a specific agent. Built-in for common agents,
  pluggable via external executables.
- **Agent variant** — an optional agent-specific version of a block,
  used when frontmatter or content is incompatible across agents.
- **Subscription** — a local binding to a coven, named `{org}-{coven}`.
  Stored in the user's config file.

## cova (Reference CLI)

`cova` is the reference implementation of the client spec, written in Go.
It applies blocks from coven repositories to the local filesystem.

Implemented commands: `add`, `apply`, `remove`, `status`.
Planned commands: `update`, `package`, `submit`,
`exporter add/remove`. See [ROADMAP.md](ROADMAP.md) for details.

Documentation: `docs/cova/`. Implementation: `cova/`.
See `cova/AGENTS.md` for Go-specific conventions and architecture.

## Editing Guidelines

- **Specs** (`docs/spec.md`, `docs/client-spec.md`) are normative documents.
  Changes to these require explicit user approval — never modify them
  as a side effect of implementation work.
- **Schemas** (`schemas/exporter/`) must stay in sync with the client spec.
- **Docs** (`docs/cova/`) describe CLI behavior. Update them when commands
  change, but keep the distinction between "implemented" and "planned".
- **Markdown** — keep lines within 120 characters
  (except code blocks and tables).
