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

- 2026-11-03 — Handoff suggests RELAI-TEAM only when two of three signals of parallel work are present (2+ owners, 2+ in-progress assignees, 2+ commit authors in 14 days) — one signal alone is normal in solo work, and the same human can appear under two author names; it is the first line cut under budget (format v0.3, PR #4).

### Pitfalls
<!-- Things that cost time once and will again: symptom → cause → what to do. -->
- `bd init` makes its own commit and edits AGENTS.md/.gitignore → use `--skip-agents --skip-hooks`, then `git reset --soft HEAD~1` and recommit.
- OpenSpec needs Node ≥ 22.12; Debian's node 18 is too old.
- Debian's Go is 1.19 while go.mod says 1.23: no `min`/`max` builtins, loop variables shared by closures (vet flags them) — CI uses 1.23.
- E2 runs `claude -p` on the logged-in account: the `cost_usd` it reports is an API-price estimate; on a subscription it uses quota, not money.
- `openspec status`/`validate` take the change name differently: `status --change "<name>" --json` but `validate "<name>" --strict` (positional, no `--change`) — passing `--change` to `validate` errors with "unknown option".
- Degraded mode's `TODO|FIXME` grep also matches harmless mentions of a file literally named `TODO.md` (e.g. "see TODO.md"), not just actionable markers — confirmed noisy-but-correct against two real repos (`patrimoine`, `rudy-ops`). Matches the spec as written; a future refinement could anchor on `TODO:`/`FIXME:` instead if this proves annoying in practice.
- 2026-11-03 — Validated this session's new features (`relai index build`/`find`, `relai new`, degraded-mode handoff) against real repos (`HOMELAB` 453 commits, `patrimoine` 141, `sport-coach`/`rudy-ops`/`freelance`/`bague`), not just synthetic fixtures: index build correctly skipped 6 real repos with no `relai.yaml` and reported why; `relai new` scaffolding a copy of `HOMELAB`'s full tree (code, a vendored binary icon, Ansible files) left every non-template file byte-identical (checksum-verified) — `Substitute`'s parse/exec-error fallback holds up on real, messy trees, not just hand-built test cases.
- 2026-11-03 — Multi-agent, multi-repo handoff simulation (10 fictional repos in `/tmp/relai-sim`, 3 fresh subagents with no shared memory): session-to-session passation on a 2-task change worked cleanly (no regression, no duplicate work, correct refusal to mark a task done when only test fakes existed); cross-repo anti-duplication worked (an agent building mobile login correctly reused an existing `auth.oidc` provider via OAuth2+PKCE instead of building its own auth). Found and fixed a real gap: in 2 of 3 subagent runs the `relai` CLI wasn't on PATH, and `AGENTS.md` never said what to do about it or that a cross-repo index exists — agents improvised (read `.relai/notes.md`/`relai.yaml`/`.relai/index.json` directly) and still got it right, but only because the data is plain files, not because the file told them to. Fixed in `internal/initrepo`'s `AGENTS.md` template (and this repo's own `AGENTS.md`): explicit "no CLI? read the files directly" fallback, explicit mention of `relai find`/`.relai/index.json` before building something new.
- Go's `flag` package stops parsing flags at the first positional argument: `relai find demo.thing --index x.json` silently ignores `--index`. CLI usage must show flags before positional args (`relai find --index x.json demo.thing`); got bitten by this writing `index build`/`find`.
- Backdate git fixture commits with `GIT_AUTHOR_DATE`/`GIT_COMMITTER_DATE` env vars when a test needs "nothing touched recently" — a freshly created `t.TempDir()` git repo's commit is always "now", so a 7-day recency window needs an explicit old date, not a bare `git init`.
- Recent goreleaser (`go install .../goreleaser/v2@latest`) refuses to build with system Go 1.19 (go.mod requires an exact `1.23.x` format Go can't satisfy) — use the Go 1.23 toolchain at `/opt/go1.23/bin` (`export PATH=/opt/go1.23/bin:$(go env GOPATH)/bin:$PATH GOFLAGS=-mod=mod`) for `go install` and `goreleaser release --snapshot --clean`.
- `goreleaser` v2's archive config uses `formats: [tar.gz]` (and `format_overrides: [{goos: windows, formats: [zip]}]`), not v1's singular `format:` key.
- A shell heredoc without quotes (`<<EOF`) runs backticked text as commands: when writing Markdown with backticks from the shell, quote the delimiter (`<<'EOF'`) or write the file with the editor tool.

### Next session
- 2026-11-03 — Open PRs: RELAI #3 (AGENTS.md no-CLI fallback; needs an approval, branch protection), #4 (handoff Team-signals hint, closes relai-xj0), #5 (positioning docs, closes relai-idg and relai-att), this housekeeping PR; RELAI-TEAM #2 (relai-mcp). Merge them, then tag the next release (v0.1.0 is already published).
- RELAI-TEAM foundation is merged and `relai-mcp` is in review. State there: `relai handoff`. `deployment-presets` is deferred on purpose (needs real requirements first).
- `team-server-mode` (docs-only change) is done but not archived: archive after the owner reviews the DESIGN §11/§12 and multi-repo §9 wording.
- Launch post drafted in `docs/posts/`; review and publish is a Beads task.
- `adoption-and-cross-repo-index` and `golden-path-template` are archived; their specs are in `openspec/specs/`.
- Test sandboxes (gone after a reboot): `/tmp/relai-sim` (10 fictional repos), `/tmp/relai-board-sim` (same, seeded with tasks and changes).
- Tooling: system Go is 1.19, use `/opt/go1.23/bin`. A `relai` binary is built at `~/.local/bin/relai`.
