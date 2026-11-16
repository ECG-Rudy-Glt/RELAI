# Proposal

## Why

Relai is git-only on purpose, but a team that outgrows it needs live claim/release across repos —
which now exists as the separate, optional [relai-team](https://github.com/ECG-Rudy-Glt/RELAI-TEAM).
Nothing tells such a team it exists. The handoff is the one document every session reads, and it
already holds the evidence of parallel work (who has tasks in progress, who commits, who owns the
repo), so it is the right place to mention it — once, only when the evidence is there.

## What Changes

- `relai handoff` adds one line to `## State` when at least two of three signals of parallel work
  are present: two or more owners in `relai.yaml`, two or more distinct assignees on in-progress
  Beads tasks, two or more distinct commit authors in the last 14 days.
- The line names the signals that fired and points to relai-team. It is never shown for a solo
  repo, and is the first thing cut when the handoff is over budget.
- Handoff format v0.2 → v0.3 (a new line kind is a minor change, per `spec/handoff.md` §7).
- No new dependency, no network call, no change to any other section.

## Impact

- **Code**: `internal/handoff`, a small exported parser in `internal/index`.
- **Docs**: `spec/handoff.md` (§4.2, §5, changelog).
- `skip_specs: true`: the format contract lives in `spec/handoff.md`, not in `openspec/specs/`.
