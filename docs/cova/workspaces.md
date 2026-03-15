# Workspaces

A workspace is a local clone of a coven repository. cova uses workspaces as a read-only local mirror of the remote state
— for reading blocks during [application][consuming-apply],
for producing patch files during [packaging][packaging], and
for applying those patches during [contribution][contributing].

---

## Location

Workspaces live under the XDG cache directory, defaulting to `~/.cache/cova/repos/`. The directory structure mirrors the
repository URL:

```
~/.cache/cova/repos/
  github.com/acme/coven-blocks/
  github.com/contoso/ai-blocks/
```

Workspaces are local copies of remote state, fully rebuildable. Deleting them is always safe; cova recreates them on
next use.

---

## One Workspace Per Repository

Workspaces are keyed by repository URL, not by subscription. When multiple subscriptions point to the same repository
(common in [monorepos][monorepo]), they share a single workspace. Each subscription's `path` field navigates to the
correct coven within the clone.

---

## How Commands Use Workspaces

> **Partial implementation:** Workspace behavior for `add` and `apply` is implemented. The workspace interactions
described for other commands below are planned but not yet available.

### Add

[`cova add`][consuming-add] clones the workspace on first use if no existing workspace exists for the repository. If
another subscription already references the same repo, the existing workspace is reused.

### Update

[`cova update`][consuming-update] fetches from the remote into the workspace before re-applying. This brings all refs up
to date in a single fetch.

### Apply

[Application][consuming-apply] reads from the workspace at each subscription's `ref`. Different subscriptions to the
same repository can track different refs — cova reads from each ref independently without requiring a checkout. The
workspace clone has all refs available locally after a fetch.

### Package

[Packaging][packaging] reads from the workspace to determine correct placement and to detect [conflicts][conflicts], but
does not modify it. The output is a git patch file written to the user's working directory.

### Contribute

[Contribution][contributing] applies the patch to the workspace on a fresh branch from the default branch, commits, and
pushes. The workspace is returned to a clean state after the operation completes.

### Remove

[`cova remove`][consuming-remove] does not touch the workspace. Applied files are deleted based on
[state tracking][state-tracking], not by reversing the workspace. The workspace persists as cache even if all
subscriptions to its repository are removed.

---

## Lifecycle

cova creates workspaces on first use (initial clone) and updates them as needed (fetch before apply or contribute).
Workspaces are not cleaned up automatically —
they persist in the cache for fast subsequent operations. Users can safely
delete the entire cache directory to reclaim space; cova will re-clone on next use.

<!-- Reference Links -->
[consuming-add]: ./consuming.md#adding-a-coven
[consuming-update]: ./consuming.md#updating
[consuming-apply]: ./consuming.md#applying
[consuming-remove]: ./consuming.md#removing-a-coven
[contributing]: ./contributing.md#git-operations
[packaging]: ./contributing.md#packaging
[conflicts]: ./contributing.md#conflict-detection
[monorepo]: ../spec.md#monorepo
[state-tracking]: ./state.md
