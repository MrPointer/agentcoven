# Framework Adapters

cova uses [adapters][adapter-protocol] to determine where blocks go for each target framework. This page covers cova's built-in adapters and how to register external ones. For the adapter protocol itself, see the [client specification][adapter-protocol].

---

## Built-in Adapters

cova ships with built-in adapters for well-known frameworks:

| Framework       | Adapter Name  | Status    |
|-----------------|---------------|-----------|
| **Claude Code** | `claude-code` | Available |
| **Cursor**      | `cursor`      | Planned   |

Built-in adapters require no registration — they're available as soon as the framework is listed in [config][configuration]. Planned adapters are not yet available; using them in config will produce an error until they are implemented.

---

## External Adapters

For frameworks not covered by built-in adapters, the community can provide external adapters following the [adapter protocol][adapter-protocol].

### Discovery

cova resolves adapter names in order:

1. **Built-in.** If the name matches a built-in adapter, use it.
2. **External.** Look for `cova-adapter-{name}` on `$PATH`.

This follows the git plugin convention — the adapter name in [config][configuration] maps directly to a discoverable executable.

### Registration

`cova adapter add <name>` registers an external adapter:

1. Verifies `cova-adapter-{name}` exists on `$PATH`.
2. Adds the name to the `frameworks` list in [config][configuration].

`cova adapter remove <name>` unregisters — removes the name from `frameworks`. It does not uninstall the executable.

<!-- Reference Links -->
[adapter-protocol]: ../client-spec.md#adapter-protocol
[configuration]: ./configuration.md
