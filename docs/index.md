# AgentCoven

AgentCoven is **agentic AI blocks as code** — a specification for managing shared AI building blocks
(skills, agents, rules, and more) through git repositories.

## Problem

Teams adopting AI coding agents lack a way to share building blocks while maintaining governance. Marketplace-style
solutions are unsuitable for internal blocks. Vendor-specific config servers introduce lock-in and do not scale well
across organizations with varying needs. Custom solutions are feasible but fragile and costly to maintain.

## Approach

AgentCoven leverages git as the backbone. Blocks are stored in a repository, changes go through pull requests, and
access is controlled by the git provider. RBAC, audit trails, compliance, and scoped ownership are already solved by the
provider — AgentCoven does not reinvent them.

## How It Works

1. A team creates a **coven repository** — a git repository structured according to the
   [AgentCoven repository specification][repo-spec]. It contains shared blocks: skills, agents, rules, and any custom
   types. Organizations may host multiple covens in a single repository using a [multi-coven][multi-coven] layout.
2. Users **subscribe** to one or more covens via a [local configuration][local-config].
3. A compliant implementation **applies** blocks from subscribed covens to the user's local filesystem, translating them
   to the format expected by their agent.

Blocks are [namespaced][naming] to ensure coexistence across covens and organizations. The user's own blocks are never
touched.

## Documentation

- **[Repository Specification][repo-spec]** — Repository structure, manifests, and block types.
- **[Client Specification][client-spec]** — Application semantics, exporter protocol, and local configuration.
- **[cova][cova]** — The reference implementation. A CLI tool that applies blocks from coven repositories to the local
  filesystem.

<!-- Reference Links -->
[repo-spec]: ./spec.md
[client-spec]: ./client-spec.md
[cova]: ./cova/index.md
[local-config]: ./client-spec.md#subscriptions
[multi-coven]: ./spec.md#multi-coven-repository
[naming]: ./spec.md#naming-convention
