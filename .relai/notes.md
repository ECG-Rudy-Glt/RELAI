### Decisions
<!-- One line each: date — decision — why. -->
- 2026-11-02 — Go, stdlib only — single binary, trivial install, easy to audit.
- 2026-11-02 — Apache-2.0 — permissive, patent grant, fits companies and foundations.
- 2026-11-02 — Easy first — individuals and small teams with only git; multi-project features are optional layers.
- 2026-11-02 — Reuse existing standards (AGENTS.md, Skills, OpenSpec, Beads, marketplaces) — they already exist (DESIGN §11).
- 2026-11-03 — Not a portal: no central service, no Backstage, no federation or SSO — the goal is collaboration between people across projects with git only (DESIGN v0.2).
- 2026-11-03 — E2 before the handoff spec — the eval decides the format.

### Pitfalls
<!-- Things that cost time once and will again: symptom → cause → what to do. -->
- `bd init` makes its own commit and edits AGENTS.md/.gitignore → use `--skip-agents --skip-hooks`, then `git reset --soft HEAD~1` and recommit.
- OpenSpec needs Node ≥ 22.12; Debian's node 18 is too old.
- Debian's Go is 1.19 while go.mod says 1.23: no `min`/`max` builtins, loop variables shared by closures (vet flags them) — CI uses 1.23.
- E2 runs `claude -p` on the logged-in account: the `cost_usd` it reports is an API-price estimate; on a subscription it uses quota, not money.
