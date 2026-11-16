# Handoff — Relai, written 2026-11-04, updated 2026-11-16

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

## 2. Where things stand (2026-11-16)

| Area | State |
|---|---|
| Release | Releases are automatic: every merge to `main` with a `feat:`/`fix:`/`perf:` commit is tagged and published by `.github/workflows/release.yml` (6 platform archives + `checksums.txt`; see the releases page for the latest); `scripts/install.sh` verifies the SHA-256 before installing and never runs `sudo` on its own. |
| CLI | Go, stdlib only, `cmd/relai`: `init` (AGENTS.md, relai.yaml, `.relai/notes.md`, and for Claude Code a `CLAUDE.md` plus a SessionStart hook; never overwrites; `--no-claude`), `handoff`, `discover`, `index build` / `find` (`--consumers`: who depends on a capability; `--json`), `index build --scan <folder>` (repeatable) and `discover --links <folder>` (find the repos of a folder and the mentions between them: evidence, never verdicts), `note add` / `note check` / `note archive` (keep `.relai/notes.md` short: dated entries, old ones listed or moved to `.relai/notes-archive.md`), `validate` (checks `relai.yaml` against `spec/relai-yaml.md`, the 100 lines of `AGENTS.md`, the presence of `.relai/notes.md`; `--strict` for CI), `tidy` (close the tasks merged commits say they close, refresh the export, delete merged branches; run by the post-merge hook), `workspace` (one budgeted overview of all indexed projects, files only), `new <template> <dir>`, `version`. Flags may come before or after the arguments. `init --hooks` adds a git pre-commit hook that keeps `.beads/issues.jsonl` current |
| Handoff | Format **v0.6** (GitHub Issues with `tasks: github` in `relai.yaml`, `--json`) (see `spec/handoff.md`, changelog at the end). Deterministic, about 2,000 tokens at most, degraded mode for repos with no harness, honest messages when `bd` or `openspec` cannot be read |
| Tests | `go test ./...`: handoff (budget, degraded mode, hygiene, large inputs), initrepo, discover, index (every `provides` list shape), scaffold (symlinks, binary files, token substitution), CLI args. CI runs gofmt, vet, test |
| Experiment E2 | `eval/README.md`. The gain is mostly in turns (1 instead of 4-30); in tokens it is 1.2x-3.4x in dollars, more in raw tokens, and nothing on repos with no tracker or notes. Do not quote the 56x figure (it is a size estimate, not an agent measurement) |
| Test report | A hostile test of both projects on 2026-11-03 (`/home/rudy/relai-test-report/`, not in git). Its findings are tasks labelled `test-report-2026-11-03` in Beads; `bd list --label test-report-2026-11-03` |
| RELAI-TEAM | The optional team layer, separate repo: [github.com/ECG-Rudy-Glt/RELAI-TEAM](https://github.com/ECG-Rudy-Glt/RELAI-TEAM). Relai never depends on it. The handoff suggests it only when two signals of parallel work are present |
| OpenSpec | One change is open, `discover-linked-repos`: its precision is measured, the test with an agent following `--prompt` is not. The others are archived; specs live in `openspec/specs/` |
| Measured 2026-11-16 | `eval/results/`: on the owner's folders, strong mentions were true 28 times out of 28, weak ones about 6 to 7 times out of 10 (poor when a repository's name is an ordinary word); `--propose` proposed 6 links, all true, but only 22% of the true ones (85% with `--min-weak 2`). `relai workspace` answered 3 questions out of 3 in one turn each; the comparison arm was spoiled by tool permissions, so "replaces the handoffs" is not shown. The five real-world checks pass; nothing was run on macOS or Windows |
| Licence | Apache-2.0 |

Not done yet: a test with a non-Claude agent; phases 2-4 of `docs/multi-repo.md` (hub,
`relai sync`, `relai check` -- those commands do not exist yet, the document is a proposal); the
launch post (`docs/posts/`) is a draft, to be checked against the measured numbers before any
publication (`relai-a7e`).

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
- **Backstage / platform engineering: later, as an export** (2026-11-16, `relai-klb`, deferred) —
  `relai.yaml` and the index exported to `catalog-info.yaml` for a platform team's catalog. One
  way, never a dependency: Relai stays git-only.
- **No coupling with memory services** (mem0 and similar) — they are not deterministic and need a
  server or a key. Relai offers `--json` output for them to read, nothing more.
- **The handoff uses the network only when the repository asks** (2026-11-16) — `tasks: github` in
  `relai.yaml`. A default-on call would have broken the "no network calls" promise of `SECURITY.md`
  and run at every session start through the hook.
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

## 4. Next steps, in order (approved by the owner 2026-11-16)

1. Phase 1 is done (2026-11-16): `spec/relai-yaml.md`, its schema, conformance cases and Python
   reader, `relai validate`, the generic session-start snippet. Only its measurement and adapter
   tasks remain, below.
2. Openness is done (2026-11-16): GitHub Issues adapter (opt-in, `tasks: github`), `--json` on
   `handoff` and `workspace`, `relai note`.
3. Measurements: done except E2 with a second agent (`relai-a7o.10`: no other agent CLI is
   installed on the owner's machine) and the `--prompt` test of `discover-linked-repos`.
4. Publication: launch post (`relai-a7e`), named adopters (`relai-lyn`).
5. Phase 2, only on real demand: hub and `relai sync`, `relai check`, CI templates (`relai-d6i.*`),
   `relai consumers` (`relai-d6i.4`), then the Backstage export (`relai-klb`).

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
