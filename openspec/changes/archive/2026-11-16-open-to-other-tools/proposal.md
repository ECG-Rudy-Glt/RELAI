## Why

The handoff's task section only worked with Beads, which few projects use; other tools could only
consume the handoff by parsing Markdown; and `.relai/notes.md`, the project's memory, already
overflows the handoff budget in this very repository with nothing to help keep it short.

## What Changes

- Handoff format v0.6: GitHub Issues as a task backend, opt-in with `tasks: github` in `relai.yaml`
  (read with `gh`, the only network call, never made without that line; `RELAI_NO_GH` disables it).
- `relai handoff --json` and `relai workspace --json`: the same facts as data, not cut to the budget.
- `relai note add | check | archive`: dated entries, old entries listed, moved to an archive file.

## Impact

`internal/handoff` (new adapter, JSON form, format version), `internal/workspace` (JSON), new
`internal/notes`, `relai.yaml` gains the optional `tasks` key (additive, same `apiVersion`).
`SECURITY.md` and the README state the one network call precisely.
