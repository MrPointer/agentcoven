# Client Specification

This document defines how a compliant client consumes and contributes to coven repositories. It is the contract for tool authors — anyone building a CLI, IDE plugin, or other implementation that interacts with covens on behalf of users.

The [repository specification][repo-spec] defines the source format. This document defines what happens on the client side.

---

## Local Configuration

The user's local configuration tracks which covens they subscribe to and which frameworks they target. The file format and location are implementation-specific, but the data model is standardized.

### Subscriptions

A subscription binds a local name to a coven within a repository:

| Field  | Required | Description                                                                                     |
|--------|----------|-------------------------------------------------------------------------------------------------|
| `name` | Yes      | Local name for this subscription. Composed as `{org}-{coven}` from the [manifest][manifest], ensuring uniqueness across subscriptions (since `org` is unique per user and coven names are unique within an org). |
| `repo` | Yes      | Repository URL.                                                                                 |
| `path` | No       | Path within the repo to the coven root. Used for [multi-coven repositories][multi-coven].       |
| `ref`  | No       | Git ref to track (branch, tag, commit). Defaults to the repo's default branch.                  |

A user can hold any number of subscriptions across any number of repositories.

### Frameworks

The configuration lists which agent frameworks to apply blocks to. Each entry must correspond to a known [adapter](#adapter-protocol) — either built-in or external.

---

## Application

Application is the process of copying blocks from a coven repository to the locations expected by the user's agent frameworks.

### Semantics

- **Copy, don't transform.** Blocks are copied as-is from the repository. The coven repository is the source of truth. No renaming or content rewriting occurs during application.
- **Adapter-driven placement.** The client delegates placement decisions to the [adapter](#adapter-protocol) for each target framework. The adapter determines where each block's files go; the client performs the actual copy and records the result.

### Scoping

A compliant client must track every file it places on disk. It must never create, modify, or delete files outside its managed set. The user's own blocks are always untouched.

### Conflict Detection

#### Conflict with User Blocks

If a coven block targets the same path as an existing file the client did not place, the client must flag the conflict and halt application for that block.

#### Conflict Between Subscriptions

If two subscriptions produce a block with the same namespaced name, the client must flag the conflict and halt application for that block. In practice this is rare — it requires two subscriptions to ship an identically named block.

Conflicts must be surfaced to the user, never silently resolved.

---

## Contributing

Contributing is the process of proposing changes to a coven repository — editing existing blocks or adding new ones.

### Semantics

- **Namespacing.** New blocks must be namespaced according to the target coven's [manifest][manifest], following the [naming convention][naming].
- **Validation.** Blocks must be validated against the relevant standard for their type (e.g., skills must comply with the [Agent Skills specification][agent-skills-spec]).
- **Default branch targeting.** Contributions always target the repository's default branch, regardless of what ref the user's subscription tracks.

### Conflict Detection

If a contribution conflicts with the current state of the default branch, the client must report the conflict and halt. The user resolves the conflict before retrying.

---

## Adapter Protocol

Adapters are functional — given a set of blocks, an adapter returns where each block should be placed. The client handles the actual file operations (copying, state tracking, conflict detection). The adapter's job is to answer "where".

### Wire Format

Adapters communicate over **stdin/stdout using JSON**. The client writes a request to the adapter's stdin and reads the response from stdout. One invocation per subscription per operation.

### External Adapter Convention

External adapters are standalone executables named `cova-adapter-{name}` (or `{tool}-adapter-{name}` for non-cova implementations) discoverable on `$PATH`. The adapter name in configuration maps directly to the executable name, following the git plugin convention.

### Apply

The client invokes the adapter once per subscription. The adapter receives the subscription's blocks grouped by type, along with the workspace path and manifest metadata. The client resolves [framework variants][framework-variants] before invocation — the `source` field in each block points to the resolved variant directory, not the block root.

**Input** (stdin):

```json
{
  "operation": "apply",
  "subscription": "platform",
  "workspace": "/home/user/.cache/cova/repos/github.com/acme/coven-blocks",
  "manifest": {
    "org": "acme",
    "coven": "platform"
  },
  "blocks": {
    "skills": [
      {
        "name": "acme-platform-code-review",
        "source": "skills/acme-platform-code-review"
      },
      {
        "name": "acme-platform-testing",
        "source": "skills/acme-platform-testing"
      }
    ],
    "rules": [
      {
        "name": "acme-platform-go-conventions",
        "source": "rules/acme-platform-go-conventions"
      }
    ]
  }
}
```

**Output** (stdout):

```json
{
  "results": [
    {
      "name": "acme-platform-code-review",
      "placements": [
        {
          "path": "/home/user/.claude/skills/acme-platform-code-review/SKILL.md",
          "source": "skills/acme-platform-code-review/SKILL.md"
        }
      ],
      "error": null
    },
    {
      "name": "acme-platform-testing",
      "placements": [
        {
          "path": "/home/user/.claude/skills/acme-platform-testing/SKILL.md",
          "source": "skills/acme-platform-testing/SKILL.md"
        },
        {
          "path": "/home/user/.claude/skills/acme-platform-testing/config.yaml",
          "source": "skills/acme-platform-testing/config.yaml"
        }
      ],
      "error": null
    },
    {
      "name": "acme-platform-go-conventions",
      "placements": null,
      "error": "unsupported block type: rules"
    }
  ]
}
```

Each result maps to one input block. A block is a directory in the coven repository, and the adapter decides which files within that directory to place and where. A single block may produce multiple placements — one per file the framework needs. The adapter inspects the block's source directory in the workspace to determine this.

For each placement, the client reads the file from the workspace at the resolved source path and copies it to the target path.

### Remove

The client invokes the adapter once per subscription being removed, so the adapter can clean up any side effects it created during apply (e.g., entries in a framework config file). The client handles deleting the block files — the adapter is only responsible for its own extras.

**Input** (stdin):

```json
{
  "operation": "remove",
  "subscription": "platform",
  "manifest": {
    "org": "acme",
    "coven": "platform"
  },
  "blocks": {
    "skills": [
      {
        "name": "acme-platform-code-review",
        "paths": [
          "/home/user/.claude/skills/acme-platform-code-review/SKILL.md"
        ]
      }
    ]
  }
}
```

**Output** (stdout):

```json
{
  "results": [
    {
      "name": "acme-platform-code-review",
      "error": null
    }
  ]
}
```

A `null` error indicates success. The client proceeds to delete the block files and update state regardless — the remove call is a notification, not a gate.

### Placement Rules

- **One result per input block.** Every block in the request must have a corresponding result.
- **No overlapping paths.** Two blocks cannot target the same path. The client must treat this as a conflict.
- **Source is relative.** The `source` field in placements is relative to the subscription's workspace root. The client resolves the full path by joining `workspace` + `source`.

### Framework Variant Resolution

Before invoking an adapter, the client resolves each block to the correct [variant][framework-variants] for that adapter:

1. If the block directory contains a `variants.yaml` file, read it. If the adapter name is listed, use the corresponding subdirectory as the variant. If the adapter name is not listed, skip the block for this adapter.
2. If the block directory does not contain a `variants.yaml` file, use the block's root content (the framework-agnostic version).

The presence of `variants.yaml` is the sole signal for variant detection. Subdirectories in blocks without `variants.yaml` are never treated as variants, regardless of their names.

A framework-specific variant must only be applied by the adapter it was authored for. The client must never pass a variant authored for one adapter to a different adapter.

The `source` field sent to the adapter reflects the resolved variant directory, not the block root.

### Trust Model

External adapters run as local executables with the user's permissions. Clients do not sandbox them. The implicit contract:

- The adapter **should not** write block files — the client handles that based on placements.
- The adapter **may** perform side effects for its framework (e.g., updating a framework config file).
- The adapter **should** clean up its own side effects on remove.

Clients cannot enforce these boundaries. Like any plugin system, trust is placed in the adapter author.

### JSON Schemas

The adapter protocol's JSON schemas are available as standalone files under [`schemas/adapter/`][schemas] for use by IDEs, validators, and adapter authors.

| Schema | File |
|--------|------|
| Apply Request | [`apply-request.schema.json`][schema-apply-req] |
| Apply Response | [`apply-response.schema.json`][schema-apply-resp] |
| Remove Request | [`remove-request.schema.json`][schema-remove-req] |
| Remove Response | [`remove-response.schema.json`][schema-remove-resp] |

<!-- Reference Links -->
[repo-spec]: ./spec.md
[manifest]: ./spec.md#root-manifest
[naming]: ./spec.md#naming-convention
[multi-coven]: ./spec.md#multi-coven-repository
[framework-variants]: ./spec.md#framework-variants
[agent-skills-spec]: https://agentskills.io/specification
[schemas]: ../schemas/adapter/
[schema-apply-req]: ../schemas/adapter/apply-request.schema.json
[schema-apply-resp]: ../schemas/adapter/apply-response.schema.json
[schema-remove-req]: ../schemas/adapter/remove-request.schema.json
[schema-remove-resp]: ../schemas/adapter/remove-response.schema.json
