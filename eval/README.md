# E2 — does the handoff let an agent start right, for fewer tokens?

The claim behind Relai's handoff: an agent that starts cold with `AGENTS.md` + `relai handoff`
picks the right next step as often as one that reads the project docs, for far fewer tokens.
E1 measured tokens only; E2 measures correctness too, with real agents.

## Protocol

**Fixtures** (`fixtures/<name>/`): small scripted projects whose correct next step is known in
advance. `build.sh <dir>` writes the repo (fixed author, dates and task IDs), `expected.json` says
which answers are right and which known wrong answers ("traps") mean, `notes.md` is the curated
`.relai/notes.md` used in condition (c) — it only restates what the docs and history already say.

| Fixture | Project | What makes it hard | Right answer |
|---|---|---|---|
| `f1-stale-docs` | Go link checker, Beads | `TODO.md` names a task that was merged since; the next task's sibling is blocked | `lc-5` worker pool |
| `f2-change-in-flight` | Go HTTP API, Beads + OpenSpec | a change is 3/5 done and in progress, an unrelated P1 chore is ready | `inv-7`, task 2.1 (429 middleware) |
| `f3-no-tracker` | static site, no tracker | long journal, the TODO list still has a dropped idea first | fix the contact form |

**Conditions**, same task prompt for all (find where the work stands, choose the single next step,
change nothing, end with `NEXT: <id> — <step>`):

- `none` — the prompt alone;
- `docs` — "start by reading the project documentation (README, docs/, TODO and journal files)";
- `relai` — the repo gets `AGENTS.md` and `.relai/notes.md` (committed), and the prompt starts with
  `AGENTS.md` and the `relai handoff` output, as a SessionStart hook would inject them.

**Agents**: Claude Code headless (`claude -p`), isolated from the user's settings, memory, hooks and
MCP servers (`--setting-sources project --strict-mcp-config`), read-only tools (`Read`, `Glob`,
`Grep`, `git`, `bd`, `openspec`, `relai`, `ls`, `cat`…; no `Edit`/`Write`/web). Working
directories are neutral (`/tmp/ws-*/NNN/project`) so paths do not leak the fixture or the condition.

**Measures** per run: correct or not (the `NEXT:` line graded against `expected.json`: accepted ID,
then trap ID, then accepted pattern, then trap pattern), total input tokens over all turns (input +
cache reads + cache writes), turns, output tokens, the API-price estimate Claude Code reports.

**Target**: (c) as correct as (b) with ≥ 10× fewer tokens net of the agent's fixed overhead.

## Run it

```sh
go run ./eval/e2 -dry                                   # build fixtures, print the prompts
go run ./eval/e2 -agents claude-haiku -runs 1           # pilot
go run ./eval/e2 -agents claude-haiku,claude-sonnet -runs 3 -parallel 4
```

Needs `git`, `bd` (Beads ≥ 1.3), `openspec` (Node ≥ 22.12) and a logged-in `claude`. Results go to
`eval/results/<timestamp>.jsonl`, a summary table is printed. On a Claude subscription the runs use
quota; `cost_usd` is the equivalent API price, not a bill.

## Known limits

- **Fixed overhead.** Every Claude Code turn re-sends ~27k tokens of system prompt and tool
  definitions; a one-turn answer cannot cost less. Ratios against short doc reads are capped by
  that floor, hence the "net of overhead" target.
- **One vendor so far.** Haiku and Sonnet are two models of the same agent. The acceptance criterion
  asks for two different agents; Codex or another CLI needs an entry in `agents` (`e2/main.go`).
- **Fixtures written by the authors of the handoff.** Traps are designed to catch stale docs; a
  fixture where the task graph itself is stale would test the opposite failure.
- **`notes.md` is curated by hand.** F3 has no tracker, so the handoff is only as good as the notes:
  it measures what disciplined notes buy, not what Relai infers.

## Results

### Pilot — 2026-11-03, Claude Haiku, 1 run per cell

| Fixture | none | docs | relai |
|---|---|---|---|
| f1-stale-docs | ✓ 681k tok, 26 turns | ✓ 194k, 9 | ✓ **27k, 1** |
| f2-change-in-flight | ✓ 682k, 29 | ✓ 387k, 18 | ✓ 361k, 12 |
| f3-no-tracker | ✓ 175k, 11 | ✓ 203k, 10 | ✓ **27k, 1** |

Findings:
1. Every condition was right: the fixtures do not discriminate on correctness yet (the agent finds
   `bd ready` or the last journal entry on its own). Harder fixtures needed.
2. Where the handoff answers the question, the agent stops after one turn: ~7× fewer tokens than
   reading the docs, all of it fixed overhead.
3. F2: with the handoff the agent still explored for 12 turns, because the handoff says
   "3/5 tasks" without naming the next one. Format change to test: show the next unchecked task of
   each active change.
