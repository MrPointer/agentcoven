# Contributing

Contributing is how blocks get into a coven repository. cova handles the mechanics — namespacing, validation, placement — so the user focuses on the block itself, not on repository conventions.

---

## Two Forms of Contribution

### Editing an Existing Block

The user modifies a block that's already applied from a coven subscription. cova identifies the block's origin via [state tracking][state-tracking] and prepares the change for the source repository.

### Proposing a New Block

The user has a local block — perhaps one they wrote for themselves — and wants to propose adding it to a coven. cova [namespaces][naming] the block according to the target coven's [manifest][manifest], validates it against the relevant standards (e.g., the [Agent Skills specification][agent-skills-spec] for skills), and places it in the correct location within the repository structure.

The user must specify which subscription to target. For existing blocks, this is inferred from state; for new blocks, it must be provided explicitly.

---

## Packaging

`cova package` prepares a block for contribution. It handles:

- **Namespacing.** New blocks are renamed to follow the `{prefix}-{team}-{block-name}` [naming convention][naming], derived from the target coven's manifest.
- **Validation.** The block is checked against the relevant standard for its type (e.g., skills must comply with the [Agent Skills specification][agent-skills-spec]).
- **Placement.** The block is mapped to the correct location in the coven repository structure.

The output is a git patch file, written to the user's current working directory. The [workspace][workspaces] is never modified — packaging is a read-and-produce operation. The patch is a standard git format patch that can be applied with `git am`, inspected, shared, or used as input to `cova submit`.

---

## Conflict Detection

The patch produced during packaging is generated against the repository's default branch. If the patch applies cleanly, packaging succeeds — even if the user's subscription tracks an older ref. Divergence alone is not a problem.

If the patch does not apply cleanly, cova reports the conflict and halts. The user must resolve the conflict before submitting — typically by updating their subscription ref, re-applying, and re-doing the edit against the current version.

---

## Git Operations

`cova submit` runs the full contribution pipeline: package, then automate the git operations. The pipeline stages are:

1. **Package** — produce the patch file. Always runs.
2. **Branch + commit** — create a branch from the repo's default branch and apply the patch.
3. **Push** — push the branch to the remote.
4. **PR** — create a pull request via the git provider's CLI if available, otherwise print a URL the user can open.

Contributions always target the repository's default branch. The branch is created from the default branch — not from the user's pinned ref. This ensures all proposals target the current state of the coven, regardless of what version the user is running locally.

Each stage after packaging is configurable. Users can opt out of automatic push (`--no-push`) or PR creation (`--no-pr`) if they prefer to handle those steps manually.

### Parallel Contributions

Because each patch is self-contained and every contribution branches from the default branch, multiple blocks can be submitted in parallel. Each contribution produces its own branch and PR, keeping changes isolated and independently reviewable. The workspace returns to a clean state after each submission, so there's no ordering dependency between them.

This is particularly useful when a user wants to propose several unrelated changes at once — each gets its own PR rather than being bundled into a single large changeset.

---

## PR Creation

cova does not implement git provider APIs. Instead, it delegates PR creation to the provider's own CLI tool:

- **GitHub** — `gh pr create`
- **GitLab** — `glab mr create`

If no provider CLI is detected, cova prints a URL the user can open to create the PR manually. This avoids API token management and provider adapter maintenance while still delivering a complete workflow for users with the standard tooling installed.

<!-- Reference Links -->
[naming]: ../spec.md#naming-convention
[manifest]: ../spec.md#root-manifest
[agent-skills-spec]: https://agentskills.io/specification
[state-tracking]: ./state.md
[workspaces]: ./workspaces.md
