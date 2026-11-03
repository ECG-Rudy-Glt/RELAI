# Handoff — Relai, end of session 1 (2026-11-03)

> For whoever picks the project up next, human or agent. Read this once; afterwards the live
> state comes from `relai handoff` (generated) and `.relai/notes.md` (decisions, pitfalls).
> Everything in this repo is written in English. No Co-Authored-By trailers in commits.

## 1. What Relai is, in three lines

A vendor-neutral layer that makes repos and organisations agent-native: a **repo contract** with a
compact, verifiable **handoff** (≤ 2,000 tokens), **check before you build** (an org capability
catalog that agents query over MCP to avoid duplicating tools), a CI-enforced **standards pack**,
and **interconnection** between repos. Full reasoning: [`DESIGN.md`](DESIGN.md).

Guiding constraint, decided with the project owner: **easy first**. It must be useful to one
person or a small team with nothing but git — one binary, no server. Organisation features are
optional layers on top.

## 2. Where things stand

| Area | State |
|---|---|
| Design | `docs/DESIGN.md` draft v0.1: problem, architecture (Mermaid), integration levels, interconnection (`relai.yaml`), standards pack, diagrams-as-models, market scan with verdict, 5-phase plan, open questions |
| Market scan | Done 2026-11-02 (DESIGN §11). Verdict: partly covered by others, the gap is real — handoff standard, duplicate detection, conformance check, cross-repo contracts |
| CLI | Go, stdlib only, `cmd/relai`: `init` (adds `AGENTS.md` or links it to `CLAUDE.md`, `relai.yaml`, `.relai/notes.md`; never overwrites), `handoff` (git + Beads + OpenSpec + notes, budget enforced by showing fewer items, `-o` to write a file), `version` |
| Tests | `internal/handoff` (basics, budget enforced, not-a-repo), `internal/initrepo` (idempotent, never overwrites, CLAUDE.md link). All pass |
| CI | GitHub Actions `ci.yml`: gofmt, vet, test — green on `main` |
| Experiment E1 | On a real ~420-commit infra repo: cold start ≈ 1,700 tokens with `AGENTS.md` + handoff vs ≈ 96,000 reading the docs naively; the Go `relai handoff` output was ≈ 490 tokens |
| Dogfooding | This repo has its own `AGENTS.md`, `relai.yaml`, `.relai/notes.md`, Beads (`relai-` prefix) and OpenSpec (`openspec/`, Claude skills in `.claude/`) |
| Licence | Apache-2.0 |

Not done yet: no spec documents in `spec/`, no eval harness (E2), no releases/binaries, no
`relai validate`/`check`, no catalog, nothing of phases 2–5.

## 3. Decisions already taken (do not reopen without a reason)

- **Go, standard library only** — single static binary, trivial install, easy to audit.
- **Apache-2.0**, public repo, English only.
- **Easy first** — every feature works for one person with only git.
- **Compose, don't rebuild** — `AGENTS.md`, MCP, Agent Skills, OpenSpec, Beads, Backstage
  descriptors (including the new `AiResource` kind and `mcp-server` API type), the Claude / Codex /
  Copilot plugin marketplaces.
- **Deterministic handoff** — built from structured sources, no LLM; budget enforced by
  truncation of lists, never by rewording. An LLM may only help curate `.relai/notes.md`.
- **Small-team catalog = a git repo of manifests + static index**, no server by default.
- **Diagrams are views generated from models** (Mermaid/D2/C4 text only); a generic system-model
  toolkit would be a sibling project, later.

## 4. Next steps, in order

Tracked in Beads — `bd ready` shows them; IDs in brackets.

1. **Handoff format spec** [`relai-a7o.2`] — `spec/handoff.md`: required sections, ordering,
   budget rules and the truncation order (tasks, then commits, then notes), the "pointer" lines
   (`bd show`, `openspec show`), versioning of the format. The current Go output is the reference.
2. **Manifest and layout spec** [`relai-a7o.1`] — `spec/relai.yaml.schema.json` + `spec/layout.md`
   (`.relai/` contents, `AGENTS.md` ↔ `CLAUDE.md`). Keep it mappable to Backstage annotations
   (`relai.dev/*`), see DESIGN §13 question 5.
3. **E2 eval harness** [`relai-a7o.7`] — the proof that matters. Plan:
   - 3+ fixture repos (small scripted git histories with Beads tasks, an OpenSpec change, a few
     notes) where the correct next step is known in advance;
   - run at least 2 different agents headless and cold, three conditions: (a) nothing,
     (b) "read the docs", (c) `AGENTS.md` + `relai handoff`;
   - measure input tokens and whether the chosen next step is right (graded against the fixture);
   - target: (c) as correct as (b) with ≥ 10× fewer tokens. Put fixtures in `eval/`.
4. **Session-start integrations** [`relai-a7o.6`] — a Claude Code SessionStart hook snippet and a
   generic shell snippet; document them in the README.
5. **Release** [`relai-a7o.8`] — goreleaser binaries for Linux/macOS/Windows + one-line install,
   so people do not need Go.
6. Then `relai validate` [`relai-a7o.5`], the GitHub Issues adapter when Beads is absent
   [`relai-a7o.3`], the "upstream changes" placeholder [`relai-a7o.4`].

Phase 0 leftovers: check name availability (Go module path is fine; npm/PyPI/domain if needed),
add `CONTRIBUTING.md` and a code of conduct before inviting contributors.

## 5. How to resume

```sh
git clone https://github.com/ECG-Rudy-Glt/RELAI && cd RELAI
go test ./... && go run ./cmd/relai handoff     # Go ≥ 1.23
bd ready                                         # Beads ≥ 1.3 (optional for reading the state)
```

Tooling notes:
- OpenSpec needs **Node ≥ 22.12**; distro Node 18 is too old.
- `bd init` makes its own commit and edits `AGENTS.md`/`.gitignore`: use
  `--skip-agents --skip-hooks`, then undo its commit and recommit cleanly. After a
  `git reset --soft`, check what is still staged before the next commit — that is how the
  licence commit (`6225eaa`) ended up also containing the Beads init files. Harmless, left as is.
- Before each commit: `gofmt -l .` must print nothing, then `go vet ./...` and `go test ./...`.

## 6. Open questions for the owner

See DESIGN §13. The ones that block phase 1: none. The ones to settle before phase 3:
`relai.yaml` standalone vs Backstage annotations, catalog hosting for teams, task leases across
agents.
