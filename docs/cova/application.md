# Application

`cova apply` reconciles the target state (files on disk) with the desired state (derived from [subscriptions][subscriptions] and [framework configuration][configuration]).

---

## Flattening

Agent frameworks typically expect blocks as flat files in a specific directory (e.g., `~/.agents/skills/`). cova flattens the coven's nested directory structure into this format.

A block at `skills/code-review/skill.md` in a coven with prefix `acme` and team `platform` becomes a single file in the framework's target directory.

## Naming Convention

Flattened files follow the pattern:

```
{prefix}--{team}--{block-name}.{ext}
```

For example:

- `acme--platform--code-review.md`
- `acme--frontend--component-patterns.md`
- `contoso--devex--ci-pipeline.md`

For [single-team repositories][single-team], the team segment comes from the subscription's `name` field.

This convention prevents collisions with the user's own blocks (no org prefix), between teams (different team segments), and between orgs (different prefixes). It's also self-describing — `ls` on the target directory immediately shows where each block came from.

## Scoping

cova only manages files that match the `{prefix}--` pattern for its known subscriptions. It will never create, modify, or delete files outside this pattern. The user's own blocks are untouched.

---

## Overlap and Conflict Detection

### Overlap with User Blocks

During `apply`, if a coven block shares a name with one of the user's existing blocks (e.g., both `writing-go-code.md` and `acme--platform--writing-go-code.md` exist in the same directory), cova surfaces an informational notice. Both files remain — semantic reconciliation is the user's responsibility, not cova's.

### Conflict Between Subscriptions

If two subscriptions produce a block with the same flattened file name, cova flags a conflict and halts application for that block until the user resolves it. In practice, this is rare — it requires two subscriptions pointing to the same team with the same block name.

Cross-org conflicts are impossible by design (different [prefixes][manifest]).

---

## State Tracking

cova maintains applied state at `~/.coven/state.yaml`:

```yaml
applied:
  - path: ~/.agents/skills/acme--platform--code-review.md
    subscription: platform-team
    source: skills/code-review/skill.md
    checksum: sha256:a1b2c3...
    applied_at: 2026-02-21T12:00:00Z

  - path: ~/.agents/skills/acme--frontend--component-patterns.md
    subscription: frontend-team
    source: skills/component-patterns/skill.md
    checksum: sha256:d4e5f6...
    applied_at: 2026-02-21T12:00:00Z
```

This enables:

- **Drift detection.** If a managed file's checksum doesn't match, cova warns or re-applies.
- **Cleanup.** On re-apply or unsubscribe, cova knows exactly which files to remove.
- **Auditability.** The user can see what's applied from where at a glance.

<!-- Reference Links -->
[subscriptions]: ../spec.md#subscriptions
[single-team]: ../spec.md#single-team-repository
[manifest]: ../spec.md#root-manifest
[configuration]: ./index.md#configuration
