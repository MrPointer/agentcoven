# Specification

This document defines how coven repositories are structured, what goes in a manifest, and how blocks are organized. It is the contract that coven maintainers must follow. For the client-side contract (application, adapters, configuration), see the [client specification][client-spec].

## Design Principles

- **Git is the backbone.** No custom registries, no proprietary storage. The git provider handles RBAC, compliance, and review workflows.
- **Vendor-agnostic by default.** Blocks are assumed portable across frameworks. When framework-specific content is unavoidable, [variants][framework-variants] allow targeted authoring without affecting other frameworks.
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

#### Framework Variants

By default, blocks are **framework-agnostic** — they are assumed to work with any agent framework. This is the common case and requires no special structure.

When a block's content is incompatible across frameworks (e.g., frontmatter fields with conflicting semantics), the block may contain **framework-specific variants**. Each variant is a subdirectory named after the target adapter:

```
skills/
  acme-platform-code-review/
    SKILL.md                    # framework-agnostic (default)

  acme-platform-deploy-pipeline/
    claude-code/
      SKILL.md                  # Claude Code variant
    opencode/
      SKILL.md                  # OpenCode variant
```

The variant directory name must match the name of the [adapter][adapter-protocol] that will consume it. This is how the client resolves which variant to use.

A block may have a mix of a root (framework-agnostic) version and framework-specific variants. It may also have only variants and no root version. The resolution order is defined in the [client specification][client-spec].

Framework-specific variants are **not portable**. A variant authored for one adapter cannot be applied by a different adapter. This is by design — the variant exists precisely because the block's content is not framework-agnostic.

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
[client-spec]: ./client-spec.md
[manifest]: #root-manifest
[team-manifest]: #team-manifest
[monorepo]: #monorepo
[single-team]: #single-team-repository
[local-config]: ./client-spec.md#subscriptions
[naming]: #naming-convention
[framework-variants]: #framework-variants
[adapter-protocol]: ./client-spec.md#adapter-protocol
[agent-skills-spec]: https://agentskills.io/specification
