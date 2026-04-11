# Configuration

cova extends the [client specification's][local-config] configuration model with implementation-specific fields. The
config file lives at `$XDG_CONFIG_HOME/cova/config.yaml` (defaulting to `~/.config/cova/config.yaml`).

This file is user-authored — edited directly or managed by commands like [`cova add`][consuming-add] and
[`cova remove`][consuming-remove].

---

## Structure

```yaml
subscriptions:
  - name: acme-platform
    repo: github.com/acme/coven-blocks
    path: covens/platform
    ref: main

  - name: acme-frontend
    repo: github.com/acme/coven-blocks
    path: covens/frontend
    ref: v2.1.0

agents:
  - claude-code
  - cursor
```

The `subscriptions` section follows the [client specification][subscriptions].

---

## Agents

| Field    | Required | Description                                                                      |
|----------|----------|----------------------------------------------------------------------------------|
| `agents` | No       | List of target agents to apply blocks to. If omitted or empty, application is a no-op with a warning. |

Every entry in `agents` must match a known [exporter][exporters]. cova validates this list before applying and rejects
unknown identifiers with a clear error — no partial application, no guessing.

Use `cova exporter add` and `cova exporter remove` to manage agent entries, or edit the file directly. See
[exporters][exporters] for details on available commands.

<!-- Reference Links -->
[local-config]: ../client-spec.md#local-configuration
[subscriptions]: ../client-spec.md#subscriptions
[exporters]: ./exporters.md
[consuming-add]: ./consuming.md#adding-a-coven
[consuming-remove]: ./consuming.md#removing-a-coven
