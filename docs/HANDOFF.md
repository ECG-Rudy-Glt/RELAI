# Handoff — Relai, end of session 3 (2026-11-03)

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
| Experiment E2 | Harness in `eval/` (4 fixtures + real repos, 3 conditions, Claude Code headless). Handoff as correct as reading the docs on every fixture, 1.6–16× fewer tokens; F4 (stale tracker) forced the format to show blocked tasks and open tasks named in commits. See `eval/README.md` |
| Market scan | Done 2026-11-02 (DESIGN §11). Verdict: partly covered by others, the gap is real — handoff standard, duplicate detection, conformance check, cross-repo contracts |
| CLI | Go, stdlib only, `cmd/relai`: `init` (adds `AGENTS.md` or links it to `CLAUDE.md`, `relai.yaml`, `.relai/notes.md`; never overwrites), `handoff` (git + Beads + OpenSpec + notes, budget enforced by showing fewer items, `-o` to write a file; **degraded mode** when none of the three is present — git-only, recently-touched-files + TODO/FIXME), `discover`, `index build` / `find` (cross-repo capability search), `new <template> <dir>` (golden-path scaffolding), `version`. Handoff **format v0.3**: adds a one-line `Team signals:` hint pointing to RELAI-TEAM when two signals of parallel work are present (PR #4) |
| Tests | `internal/handoff` (basics, budget enforced, not-a-repo, degraded mode: git section unchanged, no Tasks/Active changes, recent files + markers, truncation), `internal/initrepo` (idempotent, never overwrites, CLAUDE.md link), `internal/discover` (`HasCoreHarness`), `internal/index` (build, clone URL, find). All pass |
| CI | GitHub Actions `ci.yml`: gofmt, vet, test — green on `main` |
| Experiment E1 | On a real ~420-commit infra repo: cold start ≈ 1,700 tokens with `AGENTS.md` + handoff vs ≈ 96,000 reading the docs naively; the Go `relai handoff` output was ≈ 490 tokens |
| Dogfooding | This repo has its own `AGENTS.md`, `relai.yaml`, `.relai/notes.md`, Beads (`relai-` prefix) and OpenSpec (`openspec/`, Claude skills in `.claude/`) |
| Licence | Apache-2.0 |
| OpenSpec | Both implemented changes are **archived** and their specs live in `openspec/specs/`: `cross-repo-index`, `git-only-handoff` (from `adoption-and-cross-repo-index`) and `golden-path-scaffold` (from `golden-path-template`). `team-server-mode` (docs-only positioning, `skip_specs`) is in review, PR #5; `handoff-team-hint` in PR #4 |
| Release | **v0.1.0 is published** (6 platform archives + checksums, 2026-11-03); `scripts/install.sh` installs it (checked). It is the current `main` as of PR #2 (golden path included); PRs #3–#5 come after it, so tag a v0.1.1/v0.2.0 when they land |
| RELAI-TEAM | The optional, separate team layer: [github.com/ECG-Rudy-Glt/RELAI-TEAM](https://github.com/ECG-Rudy-Glt/RELAI-TEAM). Foundation merged (live cross-repo board, atomic claim/release, token auth, Docker); `relai-mcp` (MCP tools for agents) in its PR #2, verified with three real Claude Code agents. Relai never depends on it; docs position it (PR #5). `deployment-presets` deferred there on purpose |

Not done yet: no `relai validate`; nothing of phases 2–4 of `docs/multi-repo.md` (hub, sync,
managed blocks, `relai check`). The cross-repo index is a *flat* `index.json` built by one
command and searched by another — not the hosted/scheduled hub design in `docs/multi-repo.md`,
which stays a separate, larger, not-yet-decided proposal. The launch post is drafted
(`docs/posts/`) but unreviewed and unpublished (`relai-a7e`).

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
- **`relai find` vs `relai check`** — the cross-repo capability search (anti-duplication) is
  `relai find <capability>`; `relai check` stays reserved for contract/drift validation
  (`relai-a7o.5`, `docs/multi-repo.md` §5). Don't conflate the two when implementing.
- **Hosted/Docker index service** — documented as a deferred, optional, off-by-default layer
  (`docs/DESIGN.md` §6); not built. Pick it up only once a team actually asks for it.
- **Session-end `/clear` automation** — considered and shelved (ADR drafted in chat, not a file):
  automating `/clear` itself is Claude-Code-specific and contradicts "vendor-neutral"; the
  vendor-neutral version (a session-end hook that refreshes the handoff before any
  compaction/clear) is the right shape *if* this is ever picked back up, mirroring
  `relai-a7o.6`. Shelved because there is no external user yet asking for it.

## 4. Next steps, in order

**Current priority — land the open PRs, then decide the next slice.**
1. Review and merge, in any order: RELAI **#3** (AGENTS.md no-CLI fallback; branch protection asks
   for an approval), **#4** (handoff Team-signals hint), **#5** (positioning docs); and
   RELAI-TEAM **#2** (`relai-mcp`). Then archive `team-server-mode` once its §11/§12/§9 wording
   is reviewed.
2. Tag the next release once those land (the release workflow runs on a `vX.Y.Z` tag).
3. Review and publish the launch post (`relai-a7e`).
4. Pick from the backlog below; nothing blocks any order.

Beads tasks not covered by that change — `bd ready` shows them; IDs in brackets:

1. **E2 eval harness** [`relai-a7o.7`] — harness, 4 fixtures and a real-repo mode in `eval/`;
   results in `eval/README.md`. Correctness target met on all fixtures with the current format;
   ≥ 10× fewer tokens only on 2 of 8 cells. Left: a second agent vendor (Codex not installed),
   fixtures written by someone else.
2. **Handoff format spec** [`relai-a7o.2`] — draft v0.1 in `spec/handoff.md`, implemented (format
   line, *Check first*, *Blocked*, `next:`, *Harness*). To close after the owner's review.
   **Brownfield adoption** [`relai-a7o.9`] — `relai discover` (+ `--prompt` to draft notes) done;
   `relai init` prints the inventory.
3. **Manifest and layout spec** [`relai-a7o.1`] — `spec/relai.yaml.schema.json` + `spec/layout.md`
   (`.relai/` contents, `AGENTS.md` ↔ `CLAUDE.md`, `conventions:` pointer).
4. **Session-start integrations** [`relai-a7o.6`] — a Claude Code SessionStart hook snippet and a
   generic shell snippet; document them in the README. (The shelved session-end `/clear` idea, see
   §3 above, would mirror this task if ever picked back up.)
5. **Multi-repo strategy, rest of it** [`relai-d6i`] — hub repo, managed blocks, `relai sync`,
   `relai check` (contract/drift — not `relai find`, which the archived change covers), CI templates,
   workspace handoff (`docs/multi-repo.md`). Only the index slice (`relai-d6i.4`'s
   `build`/`find` half) exists; the rest is still just a proposal. Review before code.
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
- Debian's Go is **1.19**; the project needs **1.23**. A toolchain is at `/opt/go1.23/bin`
  (`export PATH=/opt/go1.23/bin:$PATH`). For a `relai` on your PATH:
  `go build -o ~/.local/bin/relai ./cmd/relai` (or use the v0.1.0 release).
- `bd init` makes its own commit and edits `AGENTS.md`/`.gitignore`: use
  `--skip-agents --skip-hooks`, then undo its commit and recommit cleanly. After a
  `git reset --soft`, check what is still staged before the next commit — that is how the
  licence commit (`6225eaa`) ended up also containing the Beads init files. Harmless, left as is.
- Before each commit: `gofmt -l .` must print nothing, then `go vet ./...` and `go test ./...`.
- `openspec` CLI: `status` and `validate` take the change name differently —
  `openspec status --change "<name>" --json` but `openspec validate "<name>" --strict`
  (positional, no `--change`). Passing `--change` to `validate` fails with "unknown option".

## 6. Open questions for the owner

See DESIGN §13. The ones that block phase 1: none. To settle before phase 2: what the project
index stores (copies of `relai.yaml` or pointers), task leases across agents.

Also open: whether team coordination features beyond RELAI-TEAM's board (claim expiry, a web
board, deployment presets) are wanted at all — they live in that repo's backlog, not here.
