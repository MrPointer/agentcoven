# Framework Adapters

Each supported agent framework has an adapter that handles the specifics of applying blocks. An adapter knows:

1. **Where** each [block type][block-types] belongs on disk.
2. **What format** is expected (file extension, content structure).
3. **How to transform** a block from its canonical format in the coven to the framework's expected format.

---

## Known Adapters

| Framework       | Skills              | Rules               | Agents              |
|-----------------|---------------------|---------------------|---------------------|
| **Claude Code** | `~/.claude/skills/` | TBD                 | `~/.claude/agents/` |
| **Cursor**      | TBD                 | `.cursor/rules/`    | TBD                 |

Where cross-framework standards exist (e.g., `~/.agents/skills/`), cova should prefer them over framework-specific paths to reduce adapter complexity.

---

## Adding Adapters

Adapters implement a common interface. At minimum, an adapter must:

- Map each [well-known block type][block-types] to a target directory.
- Transform block content from canonical format to the framework's expected format.

The adapter interface will be defined in detail when CLI implementation begins.

<!-- Reference Links -->
[block-types]: ../spec.md#block-types
