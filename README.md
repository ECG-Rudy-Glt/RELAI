# Relai

[![CI](https://github.com/ECG-Rudy-Glt/RELAI/actions/workflows/ci.yml/badge.svg)](https://github.com/ECG-Rudy-Glt/RELAI/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ECG-Rudy-Glt/RELAI.svg)](https://pkg.go.dev/github.com/ECG-Rudy-Glt/RELAI)
[![Licence: Apache 2.0](https://img.shields.io/badge/licence-Apache%202.0-blue.svg)](LICENSE)

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

## How it fits together

```mermaid
flowchart TB
  subgraph repo["Repo contract (every repo)"]
    agents["AGENTS.md (L0, ≤100 lines)"]
    manifest["relai.yaml (provides / consumes)"]
    notes[".relai/notes.md (decisions, pitfalls)"]
    specs["openspec/ + .beads/ (adapters, optional)"]
  end
  handoff["relai handoff (L1, ≤2k tokens, generated;<br/>degraded mode if none of the above exists)"]
  repo --> handoff
  conv["Shared conventions repo (versioned, planned)"] -. "referenced by relai.yaml" .-> repo
  index["index.json (relai.yaml of every known project)"]
  repo -- "relai index build" --> index
  person(["Person B + their agent"]) -- "git pull, read L0 + L1, then bd ready" --> repo
  person -- "relai find: check before you build" --> index
  tpl["Golden path template (planned)"] -. "new repo, pre-wired" .-> repo
```

Solid arrows are shipped; dashed ones are planned (shared conventions, golden path — see
[`docs/DESIGN.md`](docs/DESIGN.md)). Everything shipped runs from a single static binary against
local git repos — no server, no Docker, no account, including the cross-repo index above.

<details>
<summary>Same diagram, as an editable <a href="docs/diagrams/architecture.drawio">draw.io file</a></summary>

![Relai architecture diagram](docs/diagrams/architecture.png)

</details>

## Quick start

Install the binary — no Go toolchain needed:

```sh
curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh | sh
```

Or download a binary directly from the [releases page](https://github.com/ECG-Rudy-Glt/RELAI/releases)
(Linux/macOS/Windows, amd64/arm64). From source:

```sh
go install github.com/ECG-Rudy-Glt/RELAI/cmd/relai@latest   # Go ≥ 1.23
```

Then:

```sh
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

## Check before you build

Before writing a new capability, check whether one of your other projects already provides it, using
nothing but each project's own `relai.yaml` — no server, no account.

```sh
# repos.txt: one local path or clone URL per line, blank lines and #comments ignored
cat > repos.txt <<'EOF'
../other-project
https://github.com/you/another-project.git
EOF

relai index build -o .relai/index.json repos.txt   # reads each repo's relai.yaml, skips what it can't
relai find --index .relai/index.json auth.oidc      # or just `relai find auth.oidc` (default path above)
```

`relai index build` reads the list, clones any clone-URL entry to a temp dir (shallow, read-only,
cleaned up after), and collects every repo's `relai.yaml` into a single `index.json`: an array of
`{"source": "<path-or-url>", "relai_yaml": {...}}`. A listed repo that is missing, not a git repo, or
has no `relai.yaml` is skipped and reported on stderr; the rest of the index is still built.

`relai find <capability>` prints every indexed repo whose `relai.yaml` lists that capability under
`provides:` (its id, source, and `summary:`), or states plainly that nothing provides it yet. If no
index has been built yet, it says so and names the `relai index build` command instead of failing
with a file-not-found error.

This only builds and reads a flat `index.json` you keep in git or share however you like; see
[`docs/multi-repo.md`](docs/multi-repo.md) for the larger (not yet built) multi-repo design this is a
first slice of.

## Licence

[Apache-2.0](LICENSE).
