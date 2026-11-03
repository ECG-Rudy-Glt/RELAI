# Tasks

## 1. Handoff on fresh clones

- [x] 1.1 Tell apart "no task backend", "readable" and "present but unreadable"; fall back to the
      `.beads/issues.jsonl` snapshot; no raw tool output; format v0.4 in code, tests and `spec/handoff.md`.
      Verified with fake `bd` scripts for: no database, bd missing, other failure (output not leaked), no
      snapshot, healthy bd unchanged; and on real fresh clones of both repos.

## 2. Claude Code onboarding

- [x] 2.1 `relai init` creates `CLAUDE.md` (import), the hook script (executable) and `settings.json`
      (valid JSON), never overwrites, tells what to add by hand, `--no-claude` opts out; verified by tests
      including a second run that creates nothing.
- [x] 2.2 The hook prints the handoff, falls back to the plain files, and exits 0 in a broken environment;
      verified by running it with a PATH that has no relai and a project directory that does not exist.
- [x] 2.3 Real sessions: `claude -p` in fresh clones of RELAI and RELAI-TEAM with no instruction about the
      project answers from the loaded handoff in one turn (the clone without the hook needed two).

## 3. Verified install

- [x] 3.1 `scripts/install.sh` verifies the SHA-256 against `checksums.txt`; verified on the real v0.1.0
      release, a tampered archive (refused, both digests shown), a missing checksum file (refused) and the
      explicit opt-out (warns, installs).

## 4. Docs and task export

- [x] 4.1 `SECURITY.md`, `CONTRIBUTING.md`, README sections; `scripts/sync-beads.sh` exports tasks without
      the owner e-mail; a fresh clone loads them with `bd bootstrap --yes`.

## Workflow follow-up

- Archive once reviewed. Not done: signed releases or attestations; reducing the three `bd` calls per handoff.
