# cova

cova is the reference implementation of the [AgentCoven specification][spec]. It's a CLI tool written in Go that applies blocks from coven repositories to the user's local filesystem, translating them to the format expected by whatever agent framework the user runs.

Think of it like chezmoi for AI building blocks: a declarative source state in git, an applied target state on disk, and a CLI that reconciles between them.

## Application

cova reconciles the user's local filesystem with the desired state derived from their subscriptions. Blocks in the coven repository are already [namespaced][naming] — application copies them to the appropriate framework locations and tracks what it manages. See [Application][application] for the full mechanics.

---

## Adapters

Framework-specific translation is handled by pluggable adapters. Each adapter maps block types to the locations and formats a specific agent framework expects. See [Adapters][adapters] for supported frameworks and how to add new ones.

---

## Configuration

cova extends the spec's [local configuration][local-config] with implementation-specific fields. The full config lives at `~/.coven/config.yaml`:

```yaml
subscriptions:
  - name: platform-team
    repo: github.com/acme/coven-blocks
    path: teams/platform
    ref: main

  - name: frontend-team
    repo: github.com/acme/coven-blocks
    path: teams/frontend
    ref: v2.1.0

frameworks:
  - claude-code
  - cursor
```

The `subscriptions` section follows the [spec][subscriptions]. The `frameworks` section is cova-specific:

| Field        | Required | Description                                                    |
|--------------|----------|----------------------------------------------------------------|
| `frameworks` | No       | List of target agent frameworks to apply blocks to. If omitted, cova detects installed frameworks or prompts the user. |

Every entry in `frameworks` must match a known [adapter][adapters]. cova validates this list before applying and rejects unknown identifiers with a clear error — no partial application, no guessing.

<!-- Reference Links -->
[spec]: ../spec.md
[naming]: ../spec.md#naming-convention
[local-config]: ../spec.md#local-configuration
[subscriptions]: ../spec.md#subscriptions
[configuration]: #configuration
[application]: ./application.md
[adapters]: ./adapters.md
