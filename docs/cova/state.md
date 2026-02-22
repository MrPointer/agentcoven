# State

cova tracks what it has applied to the user's filesystem in a local SQLite database. This is how cova knows which files it manages, where they came from, and whether they've changed since last apply.

---

## Location

The state database lives at `$XDG_DATA_HOME/cova/state.db`, defaulting to `~/.local/share/cova/state.db`.

This follows the XDG Base Directory convention for application-managed persistent data — distinct from [configuration][configuration] (user-authored, under `$XDG_CONFIG_HOME`) and [workspaces][workspaces] (rebuildable cache, under `$XDG_CACHE_HOME`).

---

## Why SQLite

State is machine-managed data, not something users edit by hand. SQLite provides:

- **Atomic writes.** No risk of corruption from a crash mid-operation.
- **Row-level updates.** Applying or removing a single block doesn't require rewriting the entire state.
- **Querying.** "Which blocks came from this subscription?" is a SQL query, not a full-file scan.
- **Built-in locking.** Safe against concurrent access without custom locking code.

Inspectability comes through `cova status` rather than reading the database directly.

---

## Schema

Each applied block is a row in the `blocks` table:

| Column         | Type   | Description                                                        |
|----------------|--------|--------------------------------------------------------------------|
| `path`         | TEXT   | Absolute path where the block was written on disk.                 |
| `subscription` | TEXT   | Name of the subscription that owns this block.                     |
| `source`       | TEXT   | Path of the block within the coven repository.                     |
| `block_type`   | TEXT   | Block type (e.g., `skills`, `rules`, `agents`).                    |
| `framework`    | TEXT   | Target framework the block was applied to (e.g., `claude-code`).   |
| `checksum`     | TEXT   | SHA-256 hash of the applied file contents.                         |

The `path` column is the primary key — each target file maps to exactly one source block.

---

## What State Enables

- **Scoping.** cova only touches files it has recorded in state. The user's own blocks are never modified or deleted.
- **Drift detection.** If a managed file's checksum no longer matches, cova knows the file was modified outside of cova and can warn or re-apply.
- **Cleanup.** On [remove][consuming-remove], cova queries state for all files belonging to a subscription and deletes exactly those.
- **Auditability.** State provides a complete record of what's applied from where — surfaced to the user via `cova status`.

---

## Deletion and Recovery

The state database is not rebuildable from scratch — if deleted, cova loses track of which files it manages. A subsequent `cova apply` will re-apply all blocks, but orphaned files from a previous apply (e.g., blocks that were since removed from the coven) will not be cleaned up.

If the database is lost, running `cova apply` followed by manually removing any unrecognized files with the coven's naming prefix restores a clean state.

<!-- Reference Links -->
[configuration]: ./configuration.md
[workspaces]: ./workspaces.md
[consuming-remove]: ./consuming.md#removing-a-coven
