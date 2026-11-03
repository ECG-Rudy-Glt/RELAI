# Handoff — Relai, end of session 1 (2026-11-03)

> For whoever picks the project up next, human or agent. Read this once; afterwards the live
> state comes from `relai handoff` (generated) and `.relai/notes.md` (decisions, pitfalls).
> Everything in this repo is written in English. No Co-Authored-By trailers in commits.

## 1. What Relai is, in three lines

A vendor-neutral layer, files in git plus one CLI, that helps people and their agents collaborate
across all their projects: a **repo contract** with a compact, verifiable **handoff** (≤ 2,000
tokens), **check before you build** (a git-based project index), **shared conventions**, **links
between projects**, and an optional **golden path** template. Full reasoning: [`DESIGN.md`](DESIGN.md).

Guiding constraint, decided with the project owner: **easy first**. It must be useful to one
person or a small team with nothing but git — one binary, no server. **Not a portal** (DESIGN v0.2):
no central service, no Backstage, no federation or SSO.

## 2. Where things stand

| Area | State |
|---|---|
| Design | `docs/DESIGN.md` draft v0.2 (refocused 2026-11-03, no portal): problem, architecture (Mermaid), adoption levels, links between projects (`relai.yaml`), shared conventions, diagrams-as-models, market scan with verdict, 4-phase plan, open questions |
| Experiment E2 | Harness in `eval/` (3 fixtures, 3 conditions, Claude Code headless). Pilot: all correct; handoff ≈ 7× fewer tokens than reading the docs on 2 fixtures, no gain on the in-flight OpenSpec change. See `eval/README.md` |
| Market scan | Done 2026-11-02 (DESIGN §11). Verdict: partly covered by others, the gap is real — handoff standard, duplicate detection, conformance check, cross-repo contracts |
| CLI | Go, stdlib only, `cmd/relai`: `init` (adds `AGENTS.md` or links it to `CLAUDE.md`, `relai.yaml`, `.relai/notes.md`; never overwrites), `handoff` (git + Beads + OpenSpec + notes, budget enforced by showing fewer items, `-o` to write a file), `version` |
| Tests | `internal/handoff` (basics, budget enforced, not-a-repo), `internal/initrepo` (idempotent, never overwrites, CLAUDE.md link). All pass |
| CI | GitHub Actions `ci.yml`: gofmt, vet, test — green on `main` |
| Experiment E1 | On a real ~420-commit infra repo: cold start ≈ 1,700 tokens with `AGENTS.md` + handoff vs ≈ 96,000 reading the docs naively; the Go `relai handoff` output was ≈ 490 tokens |
| Dogfooding | This repo has its own `AGENTS.md`, `relai.yaml`, `.relai/notes.md`, Beads (`relai-` prefix) and OpenSpec (`openspec/`, Claude skills in `.claude/`) |
| Licence | Apache-2.0 |

Not done yet: no spec documents in `spec/`, no releases/binaries, no `relai validate`, no project
index, nothing of phases 2–4.

## 3. Decisions already taken (do not reopen without a reason)

- **Go, standard library only** — single static binary, trivial install, easy to audit.
- **Apache-2.0**, public repo, English only.
- **Easy first** — every feature works for one person with only git.
- **Compose, don't rebuild** — `AGENTS.md`, Agent Skills, OpenSpec, Beads, the Claude / Codex /
  Copilot plugin marketplaces.
- **Not a portal** — collaboration between people across projects, with git only. No central
  service, no Backstage compatibility, no federation, no SSO access control.
- **Deterministic handoff** — built from structured sources, no LLM; budget enforced by
  truncation of lists, never by rewording. An LLM may only help curate `.relai/notes.md`.
- **Project index = a git repo of `relai.yaml` files + static index**, queried by the CLI; an
  optional local MCP server, no hosting.
- **Shared conventions = a versioned git repo** that `relai.yaml` points to; no signed policy packs.
- **E2 before the handoff spec** — the eval decides the format.
- **Diagrams are views generated from models** (Mermaid/D2/C4 text only); a generic system-model
  toolkit would be a sibling project, later.

## 4. Next steps, in order

Tracked in Beads — `bd ready` shows them; IDs in brackets.

1. **E2 eval harness** [`relai-a7o.7`] — in progress, harness and 3 fixtures in `eval/`. Left:
   harder fixtures (the pilot was too easy: every condition was right), the handoff improvement the
   pilot asks for (name the next unchecked task of an active OpenSpec change), a second agent
   vendor (Codex not installed), net-of-overhead token figures. Protocol in `eval/README.md`.
2. **Handoff format spec** [`relai-a7o.2`] — `spec/handoff.md`: required sections, ordering,
   budget rules and the truncation order (tasks, then commits, then notes), the "pointer" lines
   (`bd show`, `openspec show`), versioning of the format. Written from what E2 shows.
3. **Manifest and layout spec** [`relai-a7o.1`] — `spec/relai.yaml.schema.json` + `spec/layout.md`
   (`.relai/` contents, `AGENTS.md` ↔ `CLAUDE.md`, `conventions:` pointer).
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

See DESIGN §13. The ones that block phase 1: none. To settle before phase 2: what the project
index stores (copies of `relai.yaml` or pointers), task leases across agents.
