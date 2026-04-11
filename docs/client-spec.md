# Client Specification

This document defines how a compliant client consumes and contributes to coven repositories. It is the contract for tool
authors — anyone building a CLI, IDE plugin, or other implementation that interacts with covens on behalf of users.

The [repository specification][repo-spec] defines the source format. This document defines what happens on the client
side.

---

## Local Configuration

The user's local configuration tracks which covens they subscribe to and which agents they target. The file format and
location are implementation-specific, but the data model is standardized.

### Subscriptions

A subscription binds a local name to a coven within a repository:

| Field  | Required | Description                                                                                     |
|--------|----------|-------------------------------------------------------------------------------------------------|
| `name` | Yes      | Local name for this subscription. Composed as `{org}-{coven}` from the [manifest][manifest], ensuring uniqueness across subscriptions (since `org` is unique per user and coven names are unique within an org). |
| `repo` | Yes      | Repository URL.                                                                                 |
| `path` | No       | Path within the repo to the coven root. Used for [multi-coven repositories][multi-coven].       |
| `ref`  | No       | Git ref to track (branch, tag, commit). Defaults to the repo's default branch.                  |

A user can hold any number of subscriptions across any number of repositories.

### Agents

The configuration lists which agents to apply blocks to. Each entry must correspond to a known
[exporter](#exporter-protocol) — either built-in or external.

When the agent list is empty or absent, the client has no exporters to invoke — application is a no-op. The client must
warn the user that no agents are configured and skip application without treating it as an error. Subscription mutations
(add, remove, update) must succeed independently of agent configuration.

---

## Application

Application is the process of copying blocks from a coven repository to the locations expected by the user's agents.

### Semantics

- **Copy, don't transform.** Blocks are copied as-is from the repository. The coven repository is the source of truth.
  No renaming or content rewriting occurs during application.
- **Exporter-driven placement.** The client delegates placement decisions to the [exporter](#exporter-protocol) for each
  target agent. The exporter determines where each block's files go; the client performs the actual copy and records the
  result.

### Scoping

A compliant client must track every file it places on disk. It must never create, modify, or delete files outside its
managed set. The user's own blocks are always untouched.

### Conflict Detection

#### Conflict with User Blocks

If a coven block targets the same path as an existing file the client did not place, the client must flag the conflict
and halt application for that block.

#### Conflict Between Subscriptions

If two subscriptions produce a block with the same namespaced name, the client must flag the conflict and halt
application for that block. In practice this is rare —
it requires two subscriptions to ship an identically named block.

Conflicts must be surfaced to the user, never silently resolved.

---

## Contributing

Contributing is the process of proposing changes to a coven repository — editing existing blocks or adding new ones.

### Semantics

- **Namespacing.** New blocks must be namespaced according to the target coven's [manifest][manifest], following the
  [naming convention][naming].
- **Validation.** Blocks must be validated against the relevant standard for their type
  (e.g., skills must comply with the [Agent Skills specification][agent-skills-spec]).
- **Default branch targeting.** Contributions always target the repository's default branch, regardless of what ref the
  user's subscription tracks.

### Conflict Detection

If a contribution conflicts with the current state of the default branch, the client must report the conflict and halt.
The user resolves the conflict before retrying.

---

## Exporter Protocol

Exporters are functional — given a set of blocks, an exporter returns where each block should be placed. The client
handles the actual file operations (copying, state tracking, conflict detection). The exporter's job is to answer
"where".

### Wire Format

Exporters communicate over **stdin/stdout using JSON**. The client writes a request to the exporter's stdin and reads
the response from stdout. One invocation per subscription per operation.

### External Exporter Convention

External exporters are standalone executables named `cova-exporter-{name}`
(or `{tool}-exporter-{name}` for non-cova implementations) discoverable on `$PATH`. The exporter name in configuration
maps directly to the executable name, following the git plugin convention.

### Info

The `info` operation is a stateless discovery query — the exporter describes itself. The client may invoke this at any
time, independent of any subscription or block context. The request carries no additional fields beyond `operation`.

**Input** (stdin):

```json
{"operation": "info"}
```

**Output** (stdout):

```json
{"name": "claude-code", "description": "Places skills and agents for Claude Code under ~/.claude/"}
```

Both `name` and `description` are required strings. If an external exporter exits non-zero or returns malformed JSON,
the client must fall back to using the binary name (with the `cova-exporter-` prefix stripped) as the name and an empty
string as the description.

### Apply

The client invokes the exporter once per subscription. The exporter receives the subscription's blocks grouped by type,
along with the workspace path and manifest metadata. The client resolves [agent variants][agent-variants] before
invocation — the `source` field in each block points to the resolved variant directory, not the block root.

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

Each result maps to one input block. A block is a directory in the coven repository, and the exporter decides which
files within that directory to place and where. A single block may produce multiple placements —
one per file the agent
needs. The exporter inspects the block's source directory in the workspace to determine this.

For each placement, the client reads the file from the workspace at the resolved source path and copies it to the target
path.

### Remove

The client invokes the exporter once per subscription being removed, so the exporter can clean up any side effects it
created during apply (e.g., entries in an agent config file).
The client handles deleting the block files — the exporter
is only responsible for its own extras.

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

A `null` error indicates success. The client proceeds to delete the block files and update state
regardless — the remove
call is a notification, not a gate.

If the local workspace for the subscription's repository is unavailable (e.g., the cache was manually deleted), the
client should skip the exporter notification and proceed directly with file and state cleanup.

### Placement Rules

- **One result per input block.** Every block in the request must have a corresponding result.
- **No overlapping paths.** Two blocks cannot target the same path. The client must treat this as a conflict.
- **Source is relative.** The `source` field in placements is relative to the subscription's workspace root. The client
  resolves the full path by joining `workspace` + `source`.

### Agent Variant Resolution

Before invoking an exporter, the client resolves each block to the correct [variant][agent-variants] for that exporter:

1. If the block directory contains a `variants.yaml` file, read it. If the exporter name is listed, use the
   corresponding subdirectory as the variant. If the exporter name is not listed, skip the block for this exporter.
2. If the block directory does not contain a `variants.yaml` file, use the block's root content
   (the agent-agnostic version).

The presence of `variants.yaml` is the sole signal for variant detection. Subdirectories in blocks without
`variants.yaml` are never treated as variants, regardless of their names.

An agent-specific variant must only be applied by the exporter it was authored for. The client must never pass a variant
authored for one exporter to a different exporter.

The `source` field sent to the exporter reflects the resolved variant directory, not the block root.

### Trust Model

External exporters run as local executables with the user's permissions. Clients do not sandbox them. The implicit
contract:

- The exporter **should not** write block files — the client handles that based on placements.
- The exporter **may** perform side effects for its agent (e.g., updating an agent config file).
- The exporter **should** clean up its own side effects on remove.

Clients cannot enforce these boundaries. Like any plugin system, trust is placed in the exporter author.

### JSON Schemas

The exporter protocol's JSON schemas are available as standalone files under [`schemas/exporter/`][schemas] for use by
IDEs, validators, and exporter authors.

| Schema | File |
|--------|------|
| Apply Request | [`apply-request.schema.json`][schema-apply-req] |
| Apply Response | [`apply-response.schema.json`][schema-apply-resp] |
| Remove Request | [`remove-request.schema.json`][schema-remove-req] |
| Remove Response | [`remove-response.schema.json`][schema-remove-resp] |
| Info Request | [`info-request.schema.json`][schema-info-req] |
| Info Response | [`info-response.schema.json`][schema-info-resp] |

<!-- Reference Links -->
[repo-spec]: ./spec.md
[manifest]: ./spec.md#root-manifest
[naming]: ./spec.md#naming-convention
[multi-coven]: ./spec.md#multi-coven-repository
[agent-variants]: ./spec.md#agent-variants
[agent-skills-spec]: https://agentskills.io/specification
[schemas]: ../schemas/exporter/
[schema-apply-req]: ../schemas/exporter/apply-request.schema.json
[schema-apply-resp]: ../schemas/exporter/apply-response.schema.json
[schema-remove-req]: ../schemas/exporter/remove-request.schema.json
[schema-remove-resp]: ../schemas/exporter/remove-response.schema.json
[schema-info-req]: ../schemas/exporter/info-request.schema.json
[schema-info-resp]: ../schemas/exporter/info-response.schema.json
