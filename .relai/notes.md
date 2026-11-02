### Decisions
<!-- One line each: date — decision — why. -->
- 2026-11-02 — Go, stdlib only — single binary, trivial install, easy to audit.
- 2026-11-02 — Apache-2.0 — permissive, patent grant, fits companies and foundations.
- 2026-11-02 — Easy first — individuals and small teams with only git; catalog/org features are optional layers.
- 2026-11-02 — Build on Backstage descriptors and existing marketplaces — they already exist (DESIGN §11).

### Pitfalls
<!-- Things that cost time once and will again: symptom → cause → what to do. -->
- `bd init` makes its own commit and edits AGENTS.md/.gitignore → use `--skip-agents --skip-hooks`, then `git reset --soft HEAD~1` and recommit.
- OpenSpec needs Node ≥ 22.12; Debian's node 18 is too old.
