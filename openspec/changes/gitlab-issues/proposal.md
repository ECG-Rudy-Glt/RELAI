## Why

The handoff reads tasks from Beads or, opt-in, from GitHub Issues. Teams on GitLab (gitlab.com or
self-hosted) had no way to get their open issues into the handoff.

## What Changes

- Handoff format v0.7: GitLab Issues as a task backend, opt-in with `tasks: gitlab` in `relai.yaml`,
  read with the official `glab` CLI (`glab issue list --output json --per-page 50`). Relai handles no
  token, no URL and no HTTP: `glab` knows the host and the credentials.
- Same guarantees as the GitHub backend: only when the repository asks and there is no `.beads/`, one
  call bounded to 8 seconds, `RELAI_NO_GH` turns it off, fixed failure reasons, control characters
  stripped, same budget and ordering, `--json` names the backend (`tasks.source: gitlab`).
- The "no task graph" hint mentions GitLab when the origin's host contains `gitlab`.
- `relai validate` and the schema accept `tasks: gitlab`.

## Impact

`internal/handoff` (the issue-list helper is shared with the GitHub backend), `internal/validate`,
`spec/handoff.md`, `spec/relai-yaml.md`, `spec/relai.schema.json`, one conformance case, `docs/usage.md`,
`docs/HANDOFF.md`, `SECURITY.md`. Not verified against a real GitLab or `glab`: tests use a fake `glab`.
