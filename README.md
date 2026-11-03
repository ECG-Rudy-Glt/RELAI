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
relai discover    # lists what the project already gives agents: skills, hooks, specs, tracker, docs
relai handoff     # prints where the work stands, in ≤ 2,000 tokens
```

**Adopting Relai in the middle of a project.** Nothing is moved or rewritten: `relai discover` finds
the existing harness (`AGENTS.md`/`CLAUDE.md`, skills, commands, hooks, MCP servers, Beads or GitHub
Issues, OpenSpec or Spec Kit, ADRs, handoff and journal docs, CI) and the handoff points to it.
`relai discover --prompt` prints instructions for your own agent to draft `.relai/notes.md`
(decisions, pitfalls, next step) from those docs; you review the draft and commit it.

The handoff format is specified in [`spec/handoff.md`](spec/handoff.md).

Point your agent at it: `AGENTS.md` tells any agent to run `relai handoff` first. With Claude Code you
can also run it from a SessionStart hook. `relai handoff` reads Beads tasks and OpenSpec changes when
they are present, and works without them.

## Licence

[Apache-2.0](LICENSE).
