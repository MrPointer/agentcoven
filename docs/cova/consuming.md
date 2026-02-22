# Consuming

Consuming is how blocks from a coven repository end up in the user's agent frameworks. cova handles subscribing, fetching, placing files where each framework expects them, and cleaning up — the user points at a coven and the blocks are ready to use.

---

## Adding a Coven

`cova add <repo>` subscribes to a coven. It reads the repository's [manifest][manifest], adds a subscription entry to the [config][configuration], clones the [workspace][workspaces] (or reuses an existing one), and applies the new subscription's blocks.

`--ref` pins a specific version (tag, branch, or commit SHA). Without it, the subscription tracks the repository's default branch.

### Monorepo Selection

For [monorepo][monorepo] repositories, the user must specify which team(s) to subscribe to. Team names can be passed as arguments:

```
cova add github.com/acme/coven-blocks platform frontend
```

If the repository is a monorepo and no team names are given, cova prompts the user to select interactively.

Each selected team becomes its own subscription entry in the config — they can be updated and removed independently.

---

## Updating

`cova update [name...]` fetches the latest state from the remote and re-applies. Without arguments, all subscriptions are updated. With names, only the specified subscriptions are updated.

Update always fetches, regardless of ref type. For pinned SHAs the fetch is effectively a no-op; for branches and tags it pulls the latest changes.

---

## Applying

`cova apply` reconciles the target state (files on disk) with the desired state derived from [subscriptions][subscriptions] and [framework configuration][configuration]. No network operations — it works entirely from what's already cloned locally.

Blocks in a coven repository are already [namespaced][naming] and standards-compliant. Application copies them from the repository to the target locations expected by the user's agent framework(s), using the appropriate [adapter][adapters].

For example, a skill at `skills/acme-platform-code-review/SKILL.md` in the coven repository is copied to `~/.agents/skills/acme-platform-code-review/SKILL.md` (or the equivalent path for the target framework).

No renaming or content rewriting occurs during application. The coven repository is the source of truth.

### Scoping

cova tracks which blocks it manages via [state tracking][state-tracking]. It will never create, modify, or delete blocks outside its managed set. The user's own blocks are untouched.

### Conflict Detection

#### Conflict with User Blocks

During `apply`, if a coven block has the same name as one of the user's existing blocks, cova flags the conflict and requires the user to resolve it before proceeding.

#### Conflict Between Subscriptions

If two subscriptions contain a block with the same namespaced name, cova flags a conflict and halts application for that block until the user resolves it. In practice, this is rare — it requires two subscriptions to ship an identically named block.

---

## Removing a Coven

`cova remove [name...]` unsubscribes from one or more covens. With names, those subscriptions are removed directly. Without arguments, cova prompts the user to select interactively.

For each removed subscription, cova:

1. Deletes the applied files belonging to that subscription, identified via [state tracking][state-tracking].
2. Removes the subscription entry from the [config][configuration].
3. Updates state.

The [workspace][workspaces] clone is not deleted — it's cache, shared across subscriptions, and harmless to keep. If no subscriptions reference the repository, the workspace persists until the user clears the cache directory.

<!-- Reference Links -->
[subscriptions]: ../spec.md#subscriptions
[naming]: ../spec.md#naming-convention
[manifest]: ../spec.md#root-manifest
[monorepo]: ../spec.md#monorepo
[configuration]: ./configuration.md
[adapters]: ./adapters.md
[workspaces]: ./workspaces.md
[state-tracking]: ./state.md
