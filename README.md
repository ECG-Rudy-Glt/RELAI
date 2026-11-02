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

**Status:** early development (v0.x, everything may change). Read the [design document](docs/DESIGN.md).

Made to be easy: one binary, no server, nothing but git. Useful on day one for a single person or a
small team; organisation-wide features (catalog, standards pack) are optional layers.

## Quick start

```sh
go install github.com/ECG-Rudy-Glt/RELAI/cmd/relai@latest   # Go ≥ 1.23

cd your-repo
relai init        # adds AGENTS.md (or links it to CLAUDE.md), relai.yaml, .relai/notes.md
relai handoff     # prints where the work stands, in ≤ 2,000 tokens
```

Point your agent at it: `AGENTS.md` tells any agent to run `relai handoff` first. With Claude Code you
can also run it from a SessionStart hook. `relai handoff` reads Beads tasks and OpenSpec changes when
they are present, and works without them.

## Licence

[Apache-2.0](LICENSE).
