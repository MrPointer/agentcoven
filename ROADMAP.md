# Roadmap

This document tracks what's been shipped and what's planned for AgentCoven and `cova`.

## Shipped

- [x] **Repository specification** — coven structure, manifests, block types, naming, monorepo layout
- [x] **Client specification** — application semantics, exporter protocol, local configuration
- [x] **`cova add`** — subscribe to a coven repository and apply its blocks
- [x] **`cova apply`** — reconcile local state with subscribed blocks (no network)
- [x] **Claude Code exporter** — built-in exporter for skills and agents
- [x] **External exporter execution** — call community exporters via the exporter protocol
- [x] **State tracking** — SQLite-backed record of applied blocks and placements
- [x] **`cova remove`** — unsubscribe from a coven and clean up placed files

## Phase 1 — Launch

The minimum feature set for a usable first release.

### Consuming

- [ ] **`cova update`** — fetch the latest from subscribed repositories and re-apply
- [ ] **`cova status`** — show subscriptions, applied blocks, and sync state

### Contributing

- [ ] **`cova package`** — namespace and validate blocks for contribution to a coven
- [ ] **`cova submit`** — create a branch, commit packaged blocks, and open a pull request

### Extensibility

- [ ] **`cova exporter add/remove`** — register and unregister external exporters

### Technical Debt

- [ ] **Persistent per-ref worktrees** — `apply` creates ephemeral temporary worktrees on each run and never cleans
  them up. Worktrees should be persistent per-ref (so subscriptions tracking different refs get their own named
  worktree), tracked, and cleaned up when no longer referenced. This affects `remove` (should clean up a ref's
  worktree when no remaining subscription uses it) and `apply` (should reuse existing worktrees instead of leaking
  new ones).

## Phase 2

- [ ] **Importing** — import blocks from agent-native formats (e.g. Claude Code → coven blocks)
- [ ] **Interactive menus** — guided multi-coven selection, first-run agent picker, and other interactive flows

## Phase 3

- [ ] **Native exporters** — export into agent-native formats (marketplace, plugins)
