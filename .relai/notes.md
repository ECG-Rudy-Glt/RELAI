### Decisions
<!-- One line each: date — decision — why. -->
- 2026-11-02 — Go, stdlib only — single binary, trivial install, easy to audit.
- 2026-11-02 — Apache-2.0 — permissive, patent grant, fits companies and foundations.
- 2026-11-02 — Easy first — individuals and small teams with only git; multi-project features are optional layers.
- 2026-11-02 — Reuse existing standards (AGENTS.md, Skills, OpenSpec, Beads, marketplaces) — they already exist (DESIGN §11).
- 2026-11-03 — Not a portal: no central service, no Backstage, no federation or SSO — the goal is collaboration between people across projects with git only (DESIGN v0.2).
- 2026-11-03 — E2 before the handoff spec — the eval decides the format.
- 2026-11-03 — `relai find` is the cross-repo capability search (anti-duplication); `relai check` stays reserved for contract/drift validation (`relai-a7o.5`) — avoid conflating the two.
- 2026-11-03 — Hosted/Docker index service: documented as deferred, optional, off-by-default in `DESIGN.md` §6 — not built, pick up only on real team demand.
- 2026-11-03 — Automating `/clear` itself: shelved — Claude-Code-specific, contradicts vendor-neutral positioning, and no external user has asked for it. The vendor-neutral shape (session-end hook refreshing the handoff before any compaction/clear) is the right one if revisited, mirroring `relai-a7o.6`.
- 2026-11-03 — relai.yaml parsing (`internal/index`) is a small hand-rolled parser for the exact subset the format uses (flat scalars, inline `[a, b]` lists, one-level `- capability: x` block lists), not a YAML library — matches "stdlib only" and the format is too constrained to justify the dependency.
- 2026-11-03 — Handoff format bumped to v0.2 for degraded mode (new `## Recently touched files` / `## Open markers` sections) — a new section is a minor version change per `spec/handoff.md` §7.

### Pitfalls
<!-- Things that cost time once and will again: symptom → cause → what to do. -->
- `bd init` makes its own commit and edits AGENTS.md/.gitignore → use `--skip-agents --skip-hooks`, then `git reset --soft HEAD~1` and recommit.
- OpenSpec needs Node ≥ 22.12; Debian's node 18 is too old.
- Debian's Go is 1.19 while go.mod says 1.23: no `min`/`max` builtins, loop variables shared by closures (vet flags them) — CI uses 1.23.
- E2 runs `claude -p` on the logged-in account: the `cost_usd` it reports is an API-price estimate; on a subscription it uses quota, not money.
- `openspec status`/`validate` take the change name differently: `status --change "<name>" --json` but `validate "<name>" --strict` (positional, no `--change`) — passing `--change` to `validate` errors with "unknown option".
- Go's `flag` package stops parsing flags at the first positional argument: `relai find demo.thing --index x.json` silently ignores `--index`. CLI usage must show flags before positional args (`relai find --index x.json demo.thing`); got bitten by this writing `index build`/`find`.
- Backdate git fixture commits with `GIT_AUTHOR_DATE`/`GIT_COMMITTER_DATE` env vars when a test needs "nothing touched recently" — a freshly created `t.TempDir()` git repo's commit is always "now", so a 7-day recency window needs an explicit old date, not a bare `git init`.
- Recent goreleaser (`go install .../goreleaser/v2@latest`) refuses to build with system Go 1.19 (go.mod requires an exact `1.23.x` format Go can't satisfy) — use the Go 1.23 toolchain at `/opt/go1.23/bin` (`export PATH=/opt/go1.23/bin:$(go env GOPATH)/bin:$PATH GOFLAGS=-mod=mod`) for `go install` and `goreleaser release --snapshot --clean`.
- `goreleaser` v2's archive config uses `formats: [tar.gz]` (and `format_overrides: [{goos: windows, formats: [zip]}]`), not v1's singular `format:` key.
