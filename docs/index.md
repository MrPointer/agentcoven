# AgentCoven

AgentCoven is **agentic AI blocks as code** — a specification for managing shared AI building blocks (skills, agents, rules, MCP configurations) through git repositories.

## Problem

Teams adopting AI coding agents lack a way to share building blocks while maintaining governance. Marketplace-style solutions are unsuitable for internal blocks. Vendor-specific config servers introduce lock-in and do not scale well across teams with varying needs. Custom solutions are feasible but fragile and costly to maintain.

## Approach

AgentCoven leverages git as the backbone. Blocks are stored in a repository, changes go through pull requests, and access is controlled by the git provider. RBAC, audit trails, compliance, and team-scoped ownership are already solved by the provider — AgentCoven does not reinvent them.

## How It Works

1. A team creates a **coven repository** — a git repository structured according to the [AgentCoven specification][spec]. It contains the team's shared blocks: skills, agents, rules, MCP configurations, and any custom types. Organizations with multiple teams may use a single repository with a [monorepo][monorepo] layout.
2. Users **subscribe** to one or more teams via a [local configuration file][local-config].
3. A compliant implementation **applies** blocks from subscribed teams to the user's local filesystem, translating them to the format expected by their agent framework.

The user's own blocks are never touched. Multiple teams and even multiple organizations coexist through prefix-based namespacing.

## Documentation

- **[Specification][spec]** — Repository structure, manifests, block types, and local configuration.
- **[cova][cova]** — The reference implementation. A CLI tool that applies blocks from coven repositories to the local filesystem.

<!-- Reference Links -->
[spec]: ./spec.md
[cova]: ./cova/index.md
[local-config]: ./spec.md#local-configuration
[monorepo]: ./spec.md#monorepo
