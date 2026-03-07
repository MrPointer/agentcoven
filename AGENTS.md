# Agents Guide

AgentCoven is an open specification for sharing AI agent building blocks (skills, rules, agents) through git repositories. The project has two parts: the specification and a reference CLI.

## Repository Layout

```
docs/             Specification and documentation
  spec.md         Repository specification — how covens are structured
  client-spec.md  Client specification — how tools consume covens
  cova/           Documentation for the reference CLI
schemas/          JSON schemas for the adapter protocol
cova/             Reference CLI implementation (Go) — not yet created
```

## Specifications

There are two distinct specs:

- **Repository spec** (`docs/spec.md`) — defines coven repository structure, manifests, block types, naming conventions, and framework variants. The contract for coven maintainers.
- **Client spec** (`docs/client-spec.md`) — defines application semantics, local configuration, conflict detection, contributing, and the adapter protocol. The contract for tool authors.

## Key Concepts

- **Coven** — a self-contained collection of shared blocks. A git repository contains one or more covens (single-coven or multi-coven layout).
- **Block** — a skill, rule, agent definition, or custom type. Each block is a directory within its type directory.
- **Adapter** — a function that maps blocks to filesystem placements for a specific agent framework. Built-in for common frameworks, pluggable via external executables.
- **Framework variant** — an optional framework-specific version of a block, used when frontmatter or content is incompatible across frameworks.

## cova (Reference CLI)

`cova` is the reference implementation of the client spec, written in Go. It applies blocks from coven repositories to the local filesystem.

Core commands: `add`, `apply`, `remove`, `status`, `package`, `submit`, `adapter add/remove`.

Documentation lives in `docs/cova/`. Implementation will live in `cova/`.
