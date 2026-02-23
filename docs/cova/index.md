# cova

cova is the reference implementation of the [client specification][client-spec]. It's a CLI tool written in Go that applies blocks from coven repositories to the user's local filesystem, translating them to the format expected by whatever agent framework the user runs.

Think of it like chezmoi for AI building blocks: a declarative source state in git, an applied target state on disk, and a CLI that reconciles between them.

---

## Consuming

Users subscribe to covens, keep them current, and remove them when no longer needed. `cova add` subscribes, `cova update` fetches and re-applies, `cova apply` reconciles locally, and `cova remove` unsubscribes and cleans up. See [Consuming][consuming] for the full workflow.

---

## Contributing

Users propose changes to a coven — editing existing blocks or adding new ones — through a two-stage workflow. `cova package` handles namespacing, validation, and placement. `cova submit` extends packaging with automated git operations and optional PR creation. See [Contributing][contributing] for the full workflow.

---

## Workspaces

cova maintains local clones of subscribed coven repositories under the XDG cache directory. These workspaces are shared across subscriptions to the same repository and used by both consume and contribute operations. See [Workspaces][workspaces] for details.

---

## Configuration

The user's subscriptions and framework preferences live in a YAML config file under `$XDG_CONFIG_HOME`. See [Configuration][configuration] for the full structure.

---

## State

cova tracks every file it manages in a local SQLite database under `$XDG_DATA_HOME`. State enables scoping (never touch the user's own files), drift detection, and clean removal. See [State][state] for the schema and storage details.

<!-- Reference Links -->
[client-spec]: ../client-spec.md
[consuming]: ./consuming.md
[contributing]: ./contributing.md
[workspaces]: ./workspaces.md
[configuration]: ./configuration.md
[state]: ./state.md
