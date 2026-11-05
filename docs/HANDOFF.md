# Handoff — Relai, written 2026-11-04

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

## 2. Where things stand (2026-11-04)

| Area | State |
|---|---|
| Release | **v0.2.0** is published (6 platform archives + `checksums.txt`); `scripts/install.sh` verifies the SHA-256 before installing and never runs `sudo` on its own. `main` is ahead of it by the fixes below once they merge |
| CLI | Go, stdlib only, `cmd/relai`: `init` (AGENTS.md, relai.yaml, `.relai/notes.md`, and for Claude Code a `CLAUDE.md` plus a SessionStart hook; never overwrites; `--no-claude`), `handoff`, `discover`, `index build` / `find`, `index build --scan <folder>` (repeatable) and `discover --links <folder>` (find the repos of a folder and the mentions between them: evidence, never verdicts), `tidy` (close the tasks merged commits say they close, refresh the export, delete merged branches; run by the post-merge hook), `workspace` (one budgeted overview of all indexed projects, files only), `new <template> <dir>`, `version`. Flags may come before or after the arguments. `init --hooks` adds a git pre-commit hook that keeps `.beads/issues.jsonl` current |
| Handoff | Format **v0.5** (see `spec/handoff.md`, changelog at the end). Deterministic, about 2,000 tokens at most, degraded mode for repos with no harness, honest messages when `bd` or `openspec` cannot be read |
| Tests | `go test ./...`: handoff (budget, degraded mode, hygiene, large inputs), initrepo, discover, index (every `provides` list shape), scaffold (symlinks, binary files, token substitution), CLI args. CI runs gofmt, vet, test |
| Experiment E2 | `eval/README.md`. The gain is mostly in turns (1 instead of 4-30); in tokens it is 1.2x-3.4x in dollars, more in raw tokens, and nothing on repos with no tracker or notes. Do not quote the 56x figure (it is a size estimate, not an agent measurement) |
| Test report | A hostile test of both projects on 2026-11-03 (`/home/rudy/relai-test-report/`, not in git). Its findings are tasks labelled `test-report-2026-11-03` in Beads; `bd list --label test-report-2026-11-03` |
| RELAI-TEAM | The optional team layer, separate repo: [github.com/ECG-Rudy-Glt/RELAI-TEAM](https://github.com/ECG-Rudy-Glt/RELAI-TEAM). Relai never depends on it. The handoff suggests it only when two signals of parallel work are present |
| OpenSpec | All changes so far are archived; specs live in `openspec/specs/` |
| Licence | Apache-2.0 |

Not done yet: `relai validate`; phases 2-4 of `docs/multi-repo.md` (hub, `relai sync`, `relai check`
-- those commands do not exist yet, the document is a proposal); a test with a non-Claude agent;
the launch post (`docs/posts/`) is a draft and must be corrected against the test report before
any publication (`relai-czl`).

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

1. Land the test-report fixes (tasks `test-report-2026-11-03`, P1 first), then tag a release.
2. Correct the launch post and the announcement against the measured numbers (`relai-czl`).
3. Then the backlog: `bd ready`. Phase 2 (`relai-d6i.*`: hub, sync, check; the workspace view exists as `relai workspace`, its richer blocks need the links between projects),
   `relai validate` (`relai-a7o.5`), the GitHub Issues adapter (`relai-a7o.3`).

## 5. How to resume

```sh
git clone https://github.com/ECG-Rudy-Glt/RELAI && cd RELAI
go test ./... && go run ./cmd/relai handoff     # Go ≥ 1.21 fetches the pinned toolchain
bd ready                                         # Beads ≥ 1.3 (optional for reading the state)
```

Tooling notes:
- OpenSpec needs **Node ≥ 22.12**; distro Node 18 is too old.
- Debian's Go is **1.19**, too old to fetch the toolchain pinned in `go.mod` (1.26.10). A Go 1.23 is at `/opt/go1.23/bin`
  (`export PATH=/opt/go1.23/bin:$PATH`); it downloads the pinned one on first use. For a `relai` on your PATH:
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
