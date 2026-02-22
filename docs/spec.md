# Specification

This document defines how coven repositories are structured, what goes in a manifest, how blocks are organized, and how users configure their subscriptions. It is the contract that any [compliant implementation][cova] must follow.

## Design Principles

- **Git is the backbone.** No custom registries, no proprietary storage. The git provider handles RBAC, compliance, and review workflows.
- **Vendor-agnostic.** Blocks are stored in a single, standard format. Implementations handle translation to each framework's expectations during application.
- **User sovereignty.** Implementations must never touch files they don't manage.
- **Standards-compliant.** Well-known block types follow their respective specifications (e.g., skills follow the [Agent Skills specification][agent-skills-spec]).
- **Extensible taxonomy.** Well-known block types are first-class, but teams can define their own.
- **No premature abstraction.** A block is its file(s). No mandatory metadata sidecars.

---

## Naming Convention

Blocks in a coven repository are namespaced to ensure coexistence when applied alongside blocks from other teams, organizations, or the user's own collection.

Block names follow the pattern:

```
{prefix}-{team}-{block-name}
```

For example, a skill named `code-review` owned by the `platform` team in the `acme` organization becomes `acme-platform-code-review`.

The `prefix` and `team` segments are derived from the [manifest][manifest]. Both must be lowercase alphanumeric strings with no hyphens, ensuring the combined name is unambiguous and compliant with standards such as the [Agent Skills specification][agent-skills-spec].

Implementations should handle this namespacing during submission (when a block is added to the coven), not during application. This keeps the coven repository as the source of truth and makes application a straightforward copy.

---

## Repository Structure

A **coven** is a self-contained collection of blocks owned by a team, with a [manifest][manifest]. A git repository contains one or more covens.

### Single-team Repository

The most common case: a single team with one coven at the repository root:

```
manifest.yaml
skills/
  acme-platform-code-review/
    SKILL.md
  acme-platform-testing/
    SKILL.md
agents/
  acme-platform-reviewer/
    agent.md
rules/
  acme-platform-go-conventions/
    rule.md
```

### Monorepo

Organizations with multiple teams may host them in a single repository. Each team directory is a self-contained coven with its own [team manifest][team-manifest]:

```
manifest.yaml
teams/
  platform/
    manifest.yaml
    skills/
      acme-platform-deployment/
        SKILL.md
    agents/
      acme-platform-oncall-helper/
        agent.md
  frontend/
    manifest.yaml
    skills/
      acme-frontend-component-patterns/
        SKILL.md
    rules/
      acme-frontend-accessibility/
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
| `prefix` | Yes      | First segment of namespaced block names. Lowercase alphanumeric only, no hyphens. Must be unique per user across all their subscriptions. |

For single-team repositories, the organization may be the team itself.

### Team Manifest

In a [monorepo][monorepo], each team directory contains its own `manifest.yaml`:

```yaml
team: platform
```

| Field  | Required | Description                                                          |
|--------|----------|----------------------------------------------------------------------|
| `team` | Yes      | Team name. Second segment of namespaced block names. Lowercase alphanumeric only, no hyphens. |

In a [single-team repository][single-team], the team segment is derived from the subscription's `name` field in the user's [local configuration][local-config].

---

## Block Types

### Well-known Types

| Type       | Directory  | Description                                                    | Standard                              |
|------------|------------|----------------------------------------------------------------|---------------------------------------|
| **Skills** | `skills/`  | Callable capabilities the agent can invoke when relevant.      | [Agent Skills spec][agent-skills-spec] |
| **Rules**  | `rules/`   | Persistent context and instructions that shape agent behavior. | —                                     |
| **Agents** | `agents/`  | Subagent definitions with specialized roles and tool access.   | —                                     |

### Custom Types

Any directory at the block-type level that is not a well-known type is treated as a custom block type. Implementations should handle custom types with basic operations (copy) without framework-specific transformation.

### Block Structure

Each block is a subdirectory within its type directory. The directory name is the block's [namespaced name][naming]. The directory contains the block's file(s) in the format required by the relevant standard.

For skills, this means following the [Agent Skills specification][agent-skills-spec]:

```
skills/
  acme-platform-code-review/    # namespaced block name
    SKILL.md                    # required by Agent Skills spec
```

The `name` field in `SKILL.md` frontmatter must match the directory name.

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

Users subscribe to teams, not organizations. A user can subscribe to any number of teams across any number of repositories. Each subscription's blocks carry distinct `{prefix}-{team}-` prefixes in their names, so they coexist without collision.

Within an organization, blocks from different teams are distinguished by the team segment. Across organizations, blocks are distinguished by the prefix. A true conflict requires two subscriptions to contain a block with the same full namespaced name. Implementations must detect and flag this.

### Example

A user subscribed to two teams from Acme and one from Contoso sees blocks applied as follows:

```
~/.agents/skills/
  writing-go-code/                         # user's own — untouched
    SKILL.md
  acme-platform-deployment/                # from acme/platform
    SKILL.md
  acme-frontend-component-patterns/        # from acme/frontend
    SKILL.md
  contoso-devex-ci-pipeline/               # from contoso/devex
    SKILL.md
```

<!-- Reference Links -->
[cova]: ./cova/index.md
[cova-config]: ./cova/index.md#configuration
[manifest]: #root-manifest
[team-manifest]: #team-manifest
[monorepo]: #monorepo
[single-team]: #single-team-repository
[local-config]: #local-configuration
[naming]: #naming-convention
[agent-skills-spec]: https://agentskills.io/specification
