# Exporters

cova uses [exporters][exporter-protocol] to determine where blocks go for each target agent. This page covers cova's
built-in exporters and how to register external ones. For the exporter protocol itself, see the
[client specification][exporter-protocol].

---

## Built-in Exporters

cova ships with built-in exporters for well-known agents:

| Agent           | Exporter Name | Status    |
|-----------------|---------------|-----------|
| **Claude Code** | `claude-code` | Available |
| **Cursor**      | `cursor`      | Planned   |

Built-in exporters require no registration — they're available as soon as the agent is listed in
[config][configuration]. Planned exporters are not yet available; using them in config will produce an error until they
are implemented.

---

## External Exporters

For agents not covered by built-in exporters, the community can provide external exporters following the
[exporter protocol][exporter-protocol].

### Discovery

cova resolves exporter names in order:

1. **Built-in.** If the name matches a built-in exporter, use it.
2. **External.** Look for `cova-exporter-{name}` on `$PATH`.

This follows the git plugin convention — the exporter name in [config][configuration] maps directly to a discoverable
executable.

### Registration

`cova exporter add <name>` registers an external exporter:

1. Verifies `cova-exporter-{name}` exists on `$PATH`.
2. Adds the name to the `agents` list in [config][configuration].

`cova exporter remove <name>` unregisters — removes the name from `agents`. It does not uninstall the executable.

<!-- Reference Links -->
[exporter-protocol]: ../client-spec.md#exporter-protocol
[configuration]: ./configuration.md
