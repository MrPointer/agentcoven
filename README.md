# AgentCoven

**Where agentic AI foundations gather.**

AgentCoven is an open specification for sharing the building blocks behind AI coding agents — skills, rules, agent definitions, and more — through git. Teams collect their foundations in a coven repository, users subscribe, and a compliant tool places everything where their agent expects it. No registries, no config servers, no vendor lock-in. Just git.

## Motivation

AI coding agents are becoming core infrastructure. Teams are writing skills, rules, and agent definitions — but there's no good way to share them internally while keeping governance intact.

- **Marketplaces** don't work for proprietary, internal blocks.
- **Vendor config servers** lock you in and don't scale across teams with different needs.
- **Shared drives and copy-paste** break the moment something changes.

Git already solves collaboration, access control, code review, and audit trails. AgentCoven builds on that instead of reinventing it.

## How it works

1. A team creates a **coven repository** — a git repo structured according to the [AgentCoven spec](docs/spec.md). It holds the team's shared blocks: skills, agents, rules, and any custom types.

2. Users **subscribe** to teams they care about via local configuration.

3. A compliant tool **applies** blocks to the user's machine, placing them where their agent expects them.

Blocks are namespaced (`{org}-{team}-{block}`) so they coexist cleanly across teams and organizations. The user's own blocks are never touched.

```
~/.agents/skills/
  my-custom-skill/              # yours — untouched
  acme-platform-code-review/    # from the platform team
  acme-frontend-accessibility/  # from the frontend team
  contoso-devex-ci-pipeline/    # from a different org entirely
```

## Agent-agnostic

A skill is a skill, whether you run Claude Code, Codex, Cursor, or something else. AgentCoven stores blocks in a standard format. [Exporters](docs/client-spec.md#exporter-protocol) handle translation to each agent's expectations — built-in for well-known agents, pluggable for everything else.

## Specifications

AgentCoven is defined by two specifications:

- **[Repository Specification](docs/spec.md)** — How coven repositories are structured: manifests, block types, naming, monorepo layout.
- **[Client Specification](docs/client-spec.md)** — How tools consume and contribute to covens: application semantics, exporter protocol, local configuration.

## Reference implementation

**[cova](docs/cova/index.md)** is the reference CLI. It applies blocks from coven repositories to the local filesystem — think [chezmoi](https://www.chezmoi.io/) for AI building blocks.

```
cova add platform-team --repo https://github.com/acme/coven-blocks
cova apply
cova status
```

## Current status

AgentCoven is in active development. The specifications are stable and `cova` can subscribe to covens and apply blocks today. See the **[Roadmap](ROADMAP.md)** for what's shipped and what's next. Contributions, feedback, and exporter implementations are welcome.

## License

[Apache 2.0](LICENSE)
