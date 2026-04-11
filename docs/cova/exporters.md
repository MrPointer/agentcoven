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

### Management

`cova exporter add <name> [names...]` adds one or more exporters to the `agents` list in [config][configuration].
Names are stored as-is with no PATH validation — if a name does not match a built-in or discoverable external exporter,
the error surfaces at `cova apply` time. Duplicate entries are silently skipped with an informational message.

When invoked with no arguments, `cova exporter add` lists available exporters (equivalent to `cova exporter list`).

`cova exporter remove <name> [names...]` removes one or more exporters from the `agents` list. Names not found in the
list produce a warning but the command exits successfully. Removing an exporter does not uninstall executables or clean
up placed files.

`cova exporter list` shows all available exporters — both built-in and external executables discovered on `$PATH`. Each
entry shows the exporter name, a short description, and a `[configured]` marker if the exporter is present in the
`agents` list. External exporters can provide their description via the `info` protocol operation; see the
[client specification][exporter-protocol] for details.

<!-- Reference Links -->
[exporter-protocol]: ../client-spec.md#exporter-protocol
[configuration]: ./configuration.md
