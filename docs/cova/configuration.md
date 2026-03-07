# Configuration

cova extends the [client specification's][local-config] configuration model with implementation-specific fields. The config file lives at `$XDG_CONFIG_HOME/cova/config.yaml` (defaulting to `~/.config/cova/config.yaml`).

This file is user-authored — edited directly or managed by commands like [`cova add`][consuming-add] and [`cova remove`][consuming-remove].

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

frameworks:
  - claude-code
  - cursor
```

The `subscriptions` section follows the [client specification][subscriptions].

---

## Frameworks

> **Not yet implemented:** The `frameworks` field is not yet supported in the config. Currently, the config only contains `subscriptions`. Framework configuration will be added alongside `cova apply`.

| Field        | Required | Description                                                    |
|--------------|----------|----------------------------------------------------------------|
| `frameworks` | No       | List of target agent frameworks to apply blocks to. If omitted, cova detects installed frameworks or prompts the user. |

Every entry in `frameworks` must match a known [adapter][adapters]. cova validates this list before applying and rejects unknown identifiers with a clear error — no partial application, no guessing.

<!-- Reference Links -->
[local-config]: ../client-spec.md#local-configuration
[subscriptions]: ../client-spec.md#subscriptions
[adapters]: ./adapters.md
[consuming-add]: ./consuming.md#adding-a-coven
[consuming-remove]: ./consuming.md#removing-a-coven
