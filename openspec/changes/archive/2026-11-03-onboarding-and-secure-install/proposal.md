# Proposal

## Why

Someone who clones a Relai project and asks Claude a question gets less than the project intends:

- Claude Code historically reads `CLAUDE.md`, not `AGENTS.md`, and nothing loads the handoff
  automatically: "run `relai handoff` first" is a request the agent may skip.
- Beads keeps its database outside git, so a fresh clone has no tasks, and `relai handoff` answered
  "Ready: 0" as if there were nothing to do.
- The installer (`curl | sh`) installed whatever it downloaded, with no integrity check.

## What Changes

- **Handoff tells the truth about unreadable tasks** (format v0.4): when `.beads/` exists but the tasks
  cannot be read here, it says why, how to load them (`bd bootstrap --yes`) and shows the snapshot kept in
  `.beads/issues.jsonl`, instead of an empty "Ready" list.
- **`relai init` wires Claude Code**: a `CLAUDE.md` that imports `AGENTS.md`, and a SessionStart hook
  (script plus `.claude/settings.json`) whose output is the handoff, with a fallback to the plain files and
  no way to fail a session. Never overwrites; `--no-claude` opts out.
- **The installer verifies the archive's SHA-256** against the release's `checksums.txt` and refuses to
  install on a mismatch or when it cannot verify.
- The task list is exported to git (`scripts/sync-beads.sh`, owner e-mail removed) so a clone can load it.
- `SECURITY.md` and `CONTRIBUTING.md`, linked from the README.

## Capabilities

### New Capabilities
- `claude-code-onboarding`: what `relai init` sets up so Claude Code starts from the project's state.
- `verified-install`: the install script checks what it downloads.

### Modified Capabilities
- `git-only-handoff`: the handoff reports when tasks cannot be read.

## Impact

- **Code**: `internal/handoff`, `internal/initrepo`, `cmd/relai`, `scripts/install.sh`.
- **Format**: handoff v0.3 → v0.4 (a new line kind, a minor bump per `spec/handoff.md` §7).
- **Docs**: README, `spec/handoff.md`, new `SECURITY.md`, `CONTRIBUTING.md`.
- **Not done**: signed releases or attestations (the installer says its limit), a real fix to make Beads
  keep its database in git.
