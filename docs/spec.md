# Specification

This document defines how coven repositories are structured, what goes in a manifest, and how blocks are organized. It is the contract that coven maintainers must follow. For the client-side contract (application, exporters, configuration), see the [client specification][client-spec].

## Design Principles

- **Git is the backbone.** No custom registries, no proprietary storage. The git provider handles RBAC, compliance, and review workflows.
- **Vendor-agnostic by default.** Blocks are assumed portable across agents. When agent-specific content is unavoidable, [variants][agent-variants] allow targeted authoring without affecting other agents.
- **User sovereignty.** Implementations must never touch files they don't manage.
- **Standards-compliant.** Well-known block types follow their respective specifications (e.g., skills follow the [Agent Skills specification][agent-skills-spec]).
- **Extensible taxonomy.** Well-known block types are first-class, but covens can define their own.
- **No premature abstraction.** A block is its file(s). No mandatory metadata sidecars.

---

## Naming Convention

Blocks in a coven repository are namespaced to ensure coexistence when applied alongside blocks from other covens, organizations, or the user's own collection.

Block names follow the pattern:

```
{org}-{coven}-{block-name}
```

For example, a skill named `code-review` in the `platform` coven of the `acme` organization becomes `acme-platform-code-review`.

Both the `org` and `coven` segments are derived from the [manifest][manifest]. Both must be lowercase alphanumeric strings, optionally containing hyphens (no leading or trailing hyphens, no consecutive hyphens). The combined name is unambiguous and compliant with standards such as the [Agent Skills specification][agent-skills-spec].

Implementations should handle this namespacing during submission (when a block is added to the coven), not during application. This keeps the coven repository as the source of truth and makes application a straightforward copy.

---

## Repository Structure

A **coven** is a self-contained collection of blocks, with a [manifest][manifest]. A git repository contains one or more covens.

### Single-coven Repository

The most common case: a single coven at the repository root:

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

### Multi-coven Repository

Organizations may host multiple covens in a single repository. Each coven directory under `covens/` is self-contained:

```
manifest.yaml
covens/
  platform/
    skills/
      acme-platform-deployment/
        SKILL.md
    agents/
      acme-platform-oncall-helper/
        agent.md
  frontend/
    skills/
      acme-frontend-component-patterns/
        SKILL.md
    rules/
      acme-frontend-accessibility/
        rule.md
```

The root `manifest.yaml` defines organization-level metadata and declares coven names via the [`covens`][manifest] field. Each listed name must correspond to a directory under `covens/`. Directories not listed in the manifest are ignored — they may contain shared utilities, templates, or other non-coven content. Git provider features (e.g., GitHub's CODEOWNERS) may be used to scope review requirements per coven directory.

---

## Manifest

### Root Manifest

Every coven repository has a `manifest.yaml` at its root.

**Single-coven repository:**

```yaml
org: acme
covens: platform
```

**Multi-coven repository:**

```yaml
org: acme
covens:
  - platform
  - frontend
```

| Field | Required | Description |
| --- | --- | --- |
| `org` | Yes | Organization name. First segment of namespaced block names. Lowercase alphanumeric, optionally containing hyphens (no leading/trailing, no consecutive). Must be unique per user across all their subscriptions. |
| `covens` | Yes | Coven name (string) or list of coven names. Each must be a valid [naming segment][naming]. When a list is provided, each name must correspond to a directory under `covens/` ([multi-coven repository][multi-coven]). A string value indicates a [single-coven repository][single-coven]. |

---

## Block Types

### Well-known Types

| Type       | Directory  | Description                                                    | Standard                              |
|------------|------------|----------------------------------------------------------------|---------------------------------------|
| **Skills** | `skills/`  | Callable capabilities the agent can invoke when relevant.      | [Agent Skills spec][agent-skills-spec] |
| **Rules**  | `rules/`   | Persistent context and instructions that shape agent behavior. | —                                     |
| **Agents** | `agents/`  | Subagent definitions with specialized roles and tool access.   | —                                     |

### Custom Types

Any directory at the block-type level that is not a well-known type is treated as a custom block type. Implementations should handle custom types with basic operations (copy) without agent-specific transformation.

### Block Structure

Each block is a subdirectory within its type directory. The directory name is the block's [namespaced name][naming]. The directory contains the block's file(s) in the format required by the relevant standard.

For skills, this means following the [Agent Skills specification][agent-skills-spec]:

```
skills/
  acme-platform-code-review/    # namespaced block name
    SKILL.md                    # required by Agent Skills spec
```

The `name` field in `SKILL.md` frontmatter must match the directory name.

#### Agent Variants

By default, blocks are **agent-agnostic** — they are assumed to work with any agent. This is the common case and requires no special structure.

When a block's content is incompatible across agents (e.g., frontmatter fields with conflicting semantics), the block may contain **agent-specific variants**. Variants are declared explicitly via a `variants.yaml` file in the block directory:

```
skills/
  acme-platform-code-review/
    SKILL.md                    # agent-agnostic (no variants.yaml)

  acme-platform-deploy-pipeline/
    variants.yaml               # declares which exporters have variants
    claude-code/
      SKILL.md                  # Claude Code variant
    opencode/
      SKILL.md                  # OpenCode variant
```

The `variants.yaml` file lists the exporters that have variants for this block:

```yaml
variants:
  - claude-code
  - opencode
```

Each entry must correspond to a subdirectory in the block directory, named after the [exporter][exporter-protocol] that will consume it.

A block with `variants.yaml` must not contain a root-level block file (e.g., `SKILL.md` at the block root). The presence of `variants.yaml` signals that the block is variant-only — agents not listed are not supported by this block. Files and directories not declared in `variants.yaml` are ignored during application, so block directories may contain auxiliary content alongside variant subdirectories.

A block without `variants.yaml` is agent-agnostic. Any subdirectories are treated as block content, never as variants — regardless of their names. This eliminates ambiguity: variant intent is always explicit.

Agent-specific variants are **not portable**. A variant authored for one exporter cannot be applied by a different exporter. This is by design — the variant exists precisely because the block's content is not agent-agnostic.

---

## Multi-coven and Multi-org

Users subscribe to covens, not organizations. A user can subscribe to any number of covens across any number of repositories. Each subscription's blocks carry distinct `{org}-{coven}-` prefixes in their names, so they coexist without collision.

Within an organization, blocks from different covens are distinguished by the coven segment. Across organizations, blocks are distinguished by the org segment. A true conflict requires two subscriptions to contain a block with the same full namespaced name. Implementations must detect and flag this.

### Example

A user subscribed to two covens from Acme and one from Contoso sees blocks applied as follows:

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
[multi-coven]: #multi-coven-repository
[single-coven]: #single-coven-repository
[local-config]: ./client-spec.md#subscriptions
[naming]: #naming-convention
[agent-variants]: #agent-variants
[exporter-protocol]: ./client-spec.md#exporter-protocol
[agent-skills-spec]: https://agentskills.io/specification
