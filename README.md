# Relai

**Pass the baton between people and their AI agents, across all their projects — without losing
context, and without rebuilding what already exists.**

Relai is an open, agent-agnostic layer of files in git plus one CLI. No server, no portal:

- a **repo contract** (`AGENTS.md`, `relai.yaml`, a generated handoff of a few hundred tokens, specs
  and tasks) so whoever pulls the repo — and their agent — knows the next steps immediately;
- **check before you build**: a project index (a plain git repo of `relai.yaml` files) to find what
  the other projects already provide before writing a duplicate;
- **shared conventions** that every repo points to, so every agent reads the same rules;
- **links between projects**: what each repo provides and consumes, with upstream changes shown in
  the handoff;
- an optional **golden path**: a project template born with all of the above.

It composes existing standards (AGENTS.md, Agent Skills, OpenSpec, Beads) instead of replacing them.

**Status:** early development (v0.x, everything may change). Read the [design document](docs/DESIGN.md);
the handoff is being evaluated with real agents in [`eval/`](eval/README.md).

Made to be easy: one binary, no server, nothing but git. Useful on day one for a single person or a
small team; the multi-project features (index, conventions) are optional layers.

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
