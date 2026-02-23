# Framework Adapters

An adapter translates blocks from a coven repository into the format and location expected by a specific agent framework. Adapters are the bridge between the framework-agnostic [coven structure][block-types] and the framework-specific filesystem layout.

Adapters are functional — given a set of blocks, an adapter returns where each block should be placed. cova handles the actual file operations (copying from the workspace, state tracking, conflict detection). The adapter's job is to answer "where".

---

## Built-in and External

cova ships with built-in adapters for well-known frameworks:

| Framework       | Adapter Name  |
|-----------------|---------------|
| **Claude Code** | `claude-code` |
| **Cursor**      | `cursor`      |

For frameworks not covered by built-in adapters, the community can provide **external adapters** — standalone executables that follow the [adapter protocol](#protocol).

### Discovery

cova resolves adapter names in order:

1. **Built-in.** If the name matches a built-in adapter, use it.
2. **External.** Look for `cova-adapter-{name}` on `$PATH`.

This follows the git plugin convention — the adapter name in [config][configuration] maps directly to a discoverable executable.

### Registration

`cova adapter add <name>` registers an external adapter:

1. Verifies `cova-adapter-{name}` exists on `$PATH`.
2. Adds the name to the `frameworks` list in [config][configuration].

`cova adapter remove <name>` unregisters — removes the name from `frameworks`. It does not uninstall the executable.

---

## Protocol

External adapters communicate with cova over **stdin/stdout using JSON**. cova writes a request to the adapter's stdin and reads the response from stdout. One invocation per subscription per operation.

### Apply

Called during [`cova apply`][consuming-apply] and [`cova update`][consuming-update]. cova invokes the adapter once per subscription — the adapter receives a single subscription's blocks and returns placement instructions.

**Input** (stdin):

```json
{
  "operation": "apply",
  "subscription": "platform-team",
  "workspace": "/home/user/.cache/cova/repos/github.com/acme/coven-blocks",
  "manifest": {
    "org": "acme",
    "prefix": "acme",
    "team": "platform"
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

For each placement, cova reads the file from the workspace at the resolved source path and copies it to the target path. If any block fails, cova can abort the subscription and move on to the next — partial failures from one subscription don't affect others.

### Remove

Called during [`cova remove`][consuming-remove]. cova invokes the adapter once per subscription being removed, so the adapter can clean up any side effects it created during apply (e.g., entries in a framework config file). cova handles deleting the block files itself — the adapter is only responsible for its own extras.

**Input** (stdin):

```json
{
  "operation": "remove",
  "subscription": "platform-team",
  "manifest": {
    "org": "acme",
    "prefix": "acme",
    "team": "platform"
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

A `null` error indicates success. cova proceeds to delete the block files and update state regardless — the remove call is a notification, not a gate.

---

## Placement Rules

The adapter returns absolute paths for each block. cova enforces the following:

- **One result per input block.** Every block in the request must have a corresponding result.
- **No overlapping paths.** Two blocks cannot target the same path. cova treats this as a conflict.
- **Source is relative.** The `source` field in placements is relative to the subscription's workspace root. cova resolves the full path by joining `workspace` + `source`.

---

## JSON Schemas

The protocol's JSON schemas are available as standalone files under [`schemas/adapter/`][schemas] for use by IDEs, validators, and adapter authors.

| Schema | File |
|--------|------|
| Apply Request | [`apply-request.schema.json`][schema-apply-req] |
| Apply Response | [`apply-response.schema.json`][schema-apply-resp] |
| Remove Request | [`remove-request.schema.json`][schema-remove-req] |
| Remove Response | [`remove-response.schema.json`][schema-remove-resp] |

---

## Trust Model

External adapters run as local executables with the user's permissions. cova does not sandbox them. The implicit contract:

- The adapter **should not** write block files — cova handles that based on placements.
- The adapter **may** perform side effects for its framework (e.g., updating a framework config file).
- The adapter **should** clean up its own side effects on remove.

cova cannot enforce these boundaries. Like any plugin system, trust is placed in the adapter author. Misbehaving adapters are the adapter's problem, not cova's.

<!-- Reference Links -->
[block-types]: ../spec.md#block-types
[configuration]: ./configuration.md
[consuming-apply]: ./consuming.md#applying
[consuming-update]: ./consuming.md#updating
[consuming-remove]: ./consuming.md#removing-a-coven
[schemas]: ../../schemas/adapter/
[schema-apply-req]: ../../schemas/adapter/apply-request.schema.json
[schema-apply-resp]: ../../schemas/adapter/apply-response.schema.json
[schema-remove-req]: ../../schemas/adapter/remove-request.schema.json
[schema-remove-resp]: ../../schemas/adapter/remove-response.schema.json
