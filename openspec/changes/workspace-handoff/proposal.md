## Why

Driving several linked repositories from one place needs one view that is honest. `relai workspace`
needs a prebuilt index, shows no next steps, and nothing said when a repository this one depends on moved
(tasks `relai-d6i.5`, `relai-a7o.4`). A first test on the real machine also showed task counts that did
not match `bd list`: unfinished epics left out silently, and a committed `.beads/issues.jsonl` four tasks behind.

## What Changes

- `relai handoff --workspace <folder> [--depth N] [--json]`: one compact block per git repository of a folder
  (symbolic links followed, duplicates once), from files and git only, within a budget (1,500 tokens).
- Each block: branch, uncommitted files, last commit, tasks with the date of the committed list (flagged
  when older than the last commit) and the epics not counted, `next:` of active changes, `Next session` notes.
- Upstream changes: for each `links:` entry that is a repository of the same folder, the commits made there
  after this repository's last commit. In the single-repository handoff too (format v0.8, section omitted when
  empty, never cut by the budget).
- The `relai workspace` counts say how many epics are not counted.

## Impact

`internal/workspace`, `internal/handoff` (format v0.8), `cmd/relai`, `spec/handoff.md`, `docs/`, README.
Hub, `sync`, `check` and the CI templates (`relai-d6i.1` to `.3`) were already built by `shared-conventions-hub`.
