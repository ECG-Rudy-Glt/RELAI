# Relai

**Pass the baton between people and their AI agents — without losing context, and without rebuilding
what already exists.**

Relai is an open, agent-agnostic layer that makes repositories and organisations *agent-native*:

- a **repo contract** (`AGENTS.md`, `relai.yaml`, a generated handoff of a few hundred tokens, specs
  and tasks) so whoever pulls the repo — and their agent — knows the next steps immediately;
- **check before you build**: an org-wide capability catalog, queryable by agents over MCP, that flags
  existing tools before a duplicate gets written;
- a versioned **standards pack** (coding, architecture, security) that agents follow and
  `relai check` enforces in CI, whatever the agent;
- **interconnection** between projects: what each repo provides and consumes, with change impact
  propagated across teams.

It composes existing standards (AGENTS.md, MCP, Agent Skills, Backstage descriptors, OpenSpec,
Beads) instead of replacing them.

**Status:** design phase. Read the [design document](docs/DESIGN.md). An early handoff prototype
lives in [`prototype/`](prototype/).

## Licence

To be decided (Apache-2.0 proposed).
