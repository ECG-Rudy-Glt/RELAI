# Tasks

## 1. Signals and line

- [x] 1.1 Export a manifest parser from `internal/index` and read `owners` for the handoff;
      verify with a test that a repo with `owners: [a, b]` yields two owners and one without yields none.
- [x] 1.2 Collect distinct commit authors (14 days) and distinct in-progress assignees; add the
      `Team signals:` line to `## State` when at least two signals fire; verify with tests for
      zero, one (no line) and two or three signals (line), using real git repos with two authors.
- [x] 1.3 Make it the first cut under budget; verify with a test that an over-budget handoff drops
      the line before any task list and a within-budget one keeps it.

## 2. Format and docs

- [x] 2.1 Bump the format to v0.3 in code, tests and `spec/handoff.md` (§4.2 line kind, §5 cut
      order, changelog); verify `go test ./...` and `go run ./eval/e2` fixtures still conform.
- [x] 2.2 Verify against real data: run `relai handoff` in RELAI-TEAM and in a solo repo.

## Workflow follow-up

- Archive once reviewed.
