# relai — instructions for AI agents

Start here:
0. Fresh clone? The tasks live in Beads, outside git: run `bd bootstrap --yes` once to load them
   from `.beads/issues.jsonl` (needs `bd`). Without it `bd list` is empty.
1. Run `relai handoff` (current state, next tasks, decisions, pitfalls — a few hundred tokens).
   `relai` not installed or not found? Read `.relai/notes.md` and `relai.yaml` directly instead —
   they are plain files, no CLI required.
2. Before building something another project might already provide, check
   `relai find <capability>` (or read a shared `.relai/index.json` directly, if your group has
   one) — don't duplicate what already exists elsewhere.
3. New to the project? Read `docs/HANDOFF.md` once.
4. Load anything else only when you need it.

## Rules
- Keep this file under 100 lines.
- Record decisions and pitfalls in `.relai/notes.md`, not in chat.
- After creating, updating or closing tasks, run `scripts/sync-beads.sh` and commit
  `.beads/issues.jsonl`: that file is the copy of the task list other people get when they clone.
- Every `bd` run rewrites ~3 MiB of its database on disk: never run it on a timer or in a loop.

## This repo
- Go, standard library only unless a dependency is clearly worth it. Single static binary.
- Before committing: `gofmt -l .` (must be empty), `go vet ./...`, `go test ./...`.
- Everything is written in English (code, docs, commits, issues).
- Commit messages: conventional style (`feat:`, `fix:`, `docs:`…), no Co-Authored-By or
  "Generated with" trailers.
- Design lives in `docs/DESIGN.md`; changes go through OpenSpec (`openspec/`), tasks through
  Beads (`bd ready`).
- Keep it easy: every feature must work for one person with only git, no server.
