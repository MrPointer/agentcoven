# Specification

This document defines how coven repositories are structured, what goes in a manifest, how blocks are organized, and how users configure their subscriptions. It is the contract that any [compliant implementation][cova] must follow.

## Design Principles

- **Git is the backbone.** No custom registries, no proprietary storage. The git provider handles RBAC, compliance, and review workflows.
- **Vendor-agnostic.** Blocks are stored in a single, standard format. Implementations handle translation to each framework's expectations during application.
- **User sovereignty.** Implementations must never touch files they don't manage.
- **Extensible taxonomy.** Well-known block types are first-class, but teams can define their own.
- **No premature abstraction.** A block is its file(s). No mandatory metadata sidecars.

---

## Repository Structure

A **coven** is a self-contained collection of blocks owned by a team, with a [manifest][manifest]. A git repository contains one or more covens.

### Single-team Repository

The most common case: a single team with one coven at the repository root:

```
manifest.yaml
skills/
  code-review/
    skill.md
  testing/
    skill.md
agents/
  reviewer/
    agent.md
rules/
  go-conventions/
    rule.md
mcp/
  internal-api/
    config.json
```

### Monorepo

Organizations with multiple teams may host them in a single repository. Each team directory is a self-contained coven with its own [team manifest][team-manifest]:

```
manifest.yaml
teams/
  platform/
    manifest.yaml
    skills/
      deployment/
        skill.md
    agents/
      oncall-helper/
        agent.md
  frontend/
    manifest.yaml
    skills/
      component-patterns/
        skill.md
    rules/
      accessibility/
        rule.md
```

The root `manifest.yaml` defines organization-level metadata (shared across all teams). Each team's `manifest.yaml` identifies the team. Git provider features (e.g., GitHub's CODEOWNERS) may be used to scope review requirements per team directory.

---

## Manifest

### Root Manifest

Every coven repository has a `manifest.yaml` at its root:

```yaml
org: acme
prefix: acme
```

| Field    | Required | Description                                                                                |
|----------|----------|--------------------------------------------------------------------------------------------|
| `org`    | Yes      | Organization name. Identifies the organization that owns the repository.                   |
| `prefix` | Yes      | Prefix used when flattening blocks to the target filesystem. Must be unique per user across all their subscriptions. |

For single-team repositories, the organization may be the team itself.

### Team Manifest

In a [monorepo][monorepo], each team directory contains its own `manifest.yaml`:

```yaml
team: platform
```

| Field  | Required | Description                                                          |
|--------|----------|----------------------------------------------------------------------|
| `team` | Yes      | Team name. Used as the second segment in the flattened file name.    |

In a [single-team repository][single-team], the team segment is derived from the subscription's `name` field in the user's [local configuration][local-config].

---

## Block Types

### Well-known Types

| Type       | Directory  | Description                                                    |
|------------|------------|----------------------------------------------------------------|
| **Skills** | `skills/`  | Callable capabilities the agent can invoke when relevant.      |
| **Rules**  | `rules/`   | Persistent context and instructions that shape agent behavior. |
| **Agents** | `agents/`  | Subagent definitions with specialized roles and tool access.   |
| **MCP**    | `mcp/`     | MCP server configurations.                                     |

### Custom Types

Any directory at the block-type level that is not a well-known type is treated as a custom block type. Implementations should handle custom types with basic operations (copy/flatten) without framework-specific transformation.

### Block Structure

Each block is a subdirectory within its type directory. The directory name is the block name. The directory contains the block's file(s) in a single, standard format — framework-specific translation happens during [application][cova].

```
skills/
  code-review/       # block name: "code-review"
    skill.md         # the block's content
```

Blocks are framework-agnostic. A skill is a skill regardless of whether the user runs Claude Code, Cursor, or any other agent framework.

---

## Local Configuration

The user's local configuration lives at `~/.coven/config.yaml`. This file is managed by the user (or by the implementation's CLI), not stored in the coven repository.

```yaml
subscriptions:
  - name: platform-team
    repo: github.com/acme/coven-blocks
    path: teams/platform
    ref: main

  - name: frontend-team
    repo: github.com/acme/coven-blocks
    path: teams/frontend
    ref: v2.1.0

  - name: devex
    repo: github.com/contoso/ai-blocks
    ref: main
```

### Subscriptions

| Field  | Required | Description                                                                      |
|--------|----------|----------------------------------------------------------------------------------|
| `name` | Yes      | Local name for this subscription. Used as the team segment for [single-team repos][single-team]. |
| `repo` | Yes      | Repository URL.                                                                  |
| `path` | No       | Path within the repo to the coven root. Used for [monorepos][monorepo].          |
| `ref`  | No       | Git ref to track (branch, tag, commit). Defaults to the repo's default branch.   |

Implementations may extend this configuration with additional fields (e.g., framework targets). See [cova's configuration][cova-config] for an example.

---

## Multi-team and Multi-org

Users subscribe to teams, not organizations. A user can subscribe to any number of teams across any number of repositories. Each subscription produces files with distinct `{prefix}--{team}--` prefixes during [application][cova], so they coexist without collision.

Within an organization, if two teams ship a block with the same name, the flattened names differ because the team segment differs. Across organizations, conflicts are impossible by design — different organizations have different [prefixes][manifest].

A true conflict requires two subscriptions pointing at the same team with the same block name. Implementations must detect and flag this.

### Example

A user subscribed to two teams from Acme and one from Contoso sees blocks applied as follows:

```
~/.agents/skills/
  writing-go-code.md                         # user's own — untouched
  acme--platform--deployment.md              # from acme/platform
  acme--frontend--component-patterns.md      # from acme/frontend
  contoso--devex--ci-pipeline.md             # from contoso/devex
```

<!-- Reference Links -->
[cova]: ./cova/index.md
[cova-config]: ./cova/index.md#configuration
[manifest]: #root-manifest
[team-manifest]: #team-manifest
[monorepo]: #monorepo
[single-team]: #single-team-repository
[local-config]: #local-configuration
