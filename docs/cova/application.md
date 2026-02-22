# Application

`cova apply` reconciles the target state (files on disk) with the desired state (derived from [subscriptions][subscriptions] and [framework configuration][configuration]).

---

## How Apply Works

Blocks in a coven repository are already [namespaced][naming] and standards-compliant. Application copies them from the repository to the target locations expected by the user's agent framework(s), using the appropriate [adapter][adapters].

For example, a skill at `skills/acme-platform-code-review/SKILL.md` in the coven repository is copied to `~/.agents/skills/acme-platform-code-review/SKILL.md` (or the equivalent path for the target framework).

No renaming or content rewriting occurs during application. The coven repository is the source of truth.

## Scoping

cova tracks which blocks it manages via [state tracking][state-tracking]. It will never create, modify, or delete blocks outside its managed set. The user's own blocks are untouched.

---

## Conflict Detection

### Conflict with User Blocks

During `apply`, if a coven block has the same name as one of the user's existing blocks, cova flags the conflict and requires the user to resolve it before proceeding.

### Conflict Between Subscriptions

If two subscriptions contain a block with the same namespaced name, cova flags a conflict and halts application for that block until the user resolves it. In practice, this is rare — it requires two subscriptions to ship an identically named block.

---

## State Tracking

cova maintains applied state at `~/.coven/state.yaml`:

```yaml
applied:
  - path: ~/.agents/skills/acme-platform-code-review/SKILL.md
    subscription: platform-team
    source: skills/acme-platform-code-review/SKILL.md
    checksum: sha256:a1b2c3...
    applied_at: 2026-02-21T12:00:00Z

  - path: ~/.agents/skills/acme-frontend-component-patterns/SKILL.md
    subscription: frontend-team
    source: skills/acme-frontend-component-patterns/SKILL.md
    checksum: sha256:d4e5f6...
    applied_at: 2026-02-21T12:00:00Z
```

This enables:

- **Drift detection.** If a managed file's checksum doesn't match, cova warns or re-applies.
- **Cleanup.** On re-apply or unsubscribe, cova knows exactly which files to remove.
- **Auditability.** The user can see what's applied from where at a glance.

<!-- Reference Links -->
[subscriptions]: ../spec.md#subscriptions
[naming]: ../spec.md#naming-convention
[manifest]: ../spec.md#root-manifest
[configuration]: ./index.md#configuration
[adapters]: ./adapters.md
[state-tracking]: #state-tracking
