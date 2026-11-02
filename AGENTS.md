# relai — instructions for AI agents

Start here:
1. Run `relai handoff` (current state, next tasks, decisions, pitfalls — a few hundred tokens).
2. Load anything else only when you need it.

## Rules
- Keep this file under 100 lines.
- Record decisions and pitfalls in `.relai/notes.md`, not in chat.

## This repo
- Go, standard library only unless a dependency is clearly worth it. Single static binary.
- Before committing: `gofmt -l .` (must be empty), `go vet ./...`, `go test ./...`.
- Everything is written in English (code, docs, commits, issues).
- Commit messages: conventional style (`feat:`, `fix:`, `docs:`…), no Co-Authored-By or
  "Generated with" trailers.
- Design lives in `docs/DESIGN.md`; changes go through OpenSpec (`openspec/`), tasks through
  Beads (`bd ready`).
- Keep it easy: every feature must work for one person with only git, no server.
