# Configuration

cova extends the spec's [local configuration][local-config] with implementation-specific fields. The config file lives at `$XDG_CONFIG_HOME/cova/config.yaml` (defaulting to `~/.config/cova/config.yaml`).

This file is user-authored — edited directly or managed by commands like [`cova add`][consuming-add] and [`cova remove`][consuming-remove].

---

## Structure

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

The `subscriptions` section follows the [spec][subscriptions].

---

## Frameworks

| Field        | Required | Description                                                    |
|--------------|----------|----------------------------------------------------------------|
| `frameworks` | No       | List of target agent frameworks to apply blocks to. If omitted, cova detects installed frameworks or prompts the user. |

Every entry in `frameworks` must match a known [adapter][adapters]. cova validates this list before applying and rejects unknown identifiers with a clear error — no partial application, no guessing.

<!-- Reference Links -->
[local-config]: ../spec.md#local-configuration
[subscriptions]: ../spec.md#subscriptions
[adapters]: ./adapters.md
[consuming-add]: ./consuming.md#adding-a-coven
[consuming-remove]: ./consuming.md#removing-a-coven
