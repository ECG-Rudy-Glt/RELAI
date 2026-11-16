## Why

A person or a team repeats the same rules, skills and CI lines in every repository, and they drift.
Phase 2 of `docs/multi-repo.md` answers it with one hub repository that the others pin and vendor.
Nothing of it is built yet: this change records the decisions so the work can start without
re-deriving them.

## What Changes

- `relai hub init <dir>`: scaffold a hub (conventions, skills, CI templates, `repos.txt`).
- `relai sync [--apply]`: vendor the hub's content, at the pinned tag, into a repository.
- `relai check [--outdated]`: `relai validate`, plus the handoff budget, plus drift from the hub.
- `conventions: <hub>@<tag>` in `relai.yaml` stops being reserved.

## Impact

New package `internal/hub`, new file `cmd/relai/hub.go`, `internal/validate` gains an "info" level,
`spec/relai-yaml.md`, `docs/multi-repo.md`, README. Closes relai-d6i.1, relai-d6i.2, relai-d6i.3.
