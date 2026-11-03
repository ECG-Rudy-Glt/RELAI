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
| `f4-stale-tracker` | Go billing service, Beads | a task was merged but never closed: `bd ready` lists it first and the real next task shows as blocked by it | `pg-6` backfill |

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
go run ./eval/e2 -repos "$HOME/some-repo,$HOME/other" -agents claude-sonnet   # real repos, graded by hand
```

`-repos` clones real repos (tracked files only, plus the Beads database), removes the remote so a
stray `git push` cannot reach the original, and hides project `.claude/settings*.json` so their
hooks do not add context. There is no known answer: the summary says "by hand". Keep those results
out of this repo (`-out` somewhere private).

Needs `git`, `bd` (Beads ≥ 1.3), `openspec` (Node ≥ 22.12) and a logged-in `claude`. Results go to
`eval/results/<timestamp>.jsonl`, a summary table is printed. On a Claude subscription the runs use
quota; `cost_usd` is the equivalent API price, not a bill.

## Known limits

- **Fixed overhead.** Every Claude Code turn re-sends ~27k tokens of system prompt and tool
  definitions; a one-turn answer cannot cost less. Ratios against short doc reads are capped by
  that floor, hence the "net of overhead" target.
- **One vendor so far.** Haiku and Sonnet are two models of the same agent. The acceptance criterion
  asks for two different agents; Codex or another CLI needs an entry in `agents` (`e2/main.go`).
- **Fixtures written by the authors of the handoff.** F1–F3 trap agents that trust stale docs;
  F4 traps the opposite, an agent that trusts a stale task graph. More fixtures from other people
  would make the eval less self-serving.
- **`notes.md` is curated by hand.** F3 has no tracker, so the handoff is only as good as the notes:
  it measures what disciplined notes buy, not what Relai infers.

## Results

### Fixtures — 2026-11-03, Claude Haiku and Sonnet, 2–3 runs per cell

Average input tokens over all turns, turns in brackets. `relai` uses the current handoff format
(next unchecked OpenSpec task, blocked tasks, open tasks named in recent commits), except F1
which ran with v0 (no difference for F1's handoff that matters).

| Fixture | Agent | none | docs | relai | docs ÷ relai |
|---|---|---|---|---|---|
| F1 stale docs | Haiku | 3/3 · 430k (21) | 3/3 · 343k (17) | 3/3 · **27k (1)** | 12.8× |
| | Sonnet | 3/3 · 129k (6) | 3/3 · 106k (4) | 3/3 · **26k (1)** | 4.1× |
| F2 change in flight | Haiku | 3/3 · 693k (30) | 3/3 · 572k (25) | 3/3 · 86k (4) | 6.7× |
| | Sonnet | 3/3 · 112k (5) | 3/3 · 141k (6) | 3/3 · **26k (1)** | 5.4× |
| F3 no tracker | Haiku | 3/3 · 183k (12) | 2/2 · 191k (10) | 3/3 · 93k (5) | 2.0× |
| | Sonnet | 3/3 · 97k (4) | 2/2 · 107k (4) | 3/3 · **26k (1)** | 4.1× |
| F4 stale tracker | Haiku | 2/2 · 750k (30) | 2/2 · 428k (20) | 2/2 · **27k (1)** | 15.9× |
| | Sonnet | 2/2 · 89k (4) | 2/2 · 104k (4) | 2/2 · 66k (3) | 1.6× |

(F3 docs: one run per model was lost when a session ended.)

### What changed the format

| Handoff version | F2 Haiku | F2 Sonnet | F4 Haiku | F4 Sonnet |
|---|---|---|---|---|
| v0: "3/5 tasks", ready and in-progress tasks only | 3/3 · 205k (7) | 3/3 · 79k (3) | — | — |
| + next unchecked task of each active change | 3/3 · 86k (4) | 3/3 · 26k (1) | **0/2** · 27k (1) | 2/2 · 80k (3) |
| + blocked tasks, open tasks named in recent commits | — | — | 2/2 · 27k (1) | 2/2 · 66k (3) |

F4 is the important one: with a stale task list, the compact handoff made Haiku confident and
wrong in one turn ("redo the migration"), where reading the docs got it right for 16× the tokens.
Listing blocked tasks and flagging open tasks that recent commits mention fixed it without
losing the savings. A handoff that is wrong is worse than no handoff, so the format must surface
contradictions between its sources, not hide them.

### Real repos — 4 of the author's projects, Sonnet, 1 run per cell

| Repo profile | none | docs | relai | Same answer in all three? |
|---|---|---|---|---|
| infra repo, ~450 commits, Beads + OpenSpec, ~2 MB of docs | 128k (4) | 101k (3) | **30k (1)** | yes |
| web app, ~140 commits, no tracker, no notes | 102k (4) | 94k (3) | 113k (4) | yes |
| consulting repo, ~45 commits, TODO + handoff doc, no tracker | 104k (4) | 105k (4) | 107k (4) | none = relai, docs differs |
| open-source app, ~60 commits, `AGENTS.md`, issues on GitHub | 248k (8) | 181k (6) | 138k (4) | relai more generic (check CI first) |

Where the repo has structured sources (task graph, specs), the handoff gives the same answer for
3–4× fewer tokens. Where it has none, the handoff is little more than `git log` and saves
nothing: `relai init` should help start `.relai/notes.md`, and the GitHub Issues adapter
(`relai-a7o.3`) matters.

### Status against the target

- Correctness: (c) ≥ (b) on every fixture with the current format, both models.
- Tokens: ≥ 10× fewer only for Haiku on F1 and F4; 1.6–6.7× elsewhere. Every Claude Code call
  costs ~26k tokens before reading anything, so a one-turn answer cannot go lower; the gain is in
  turns (1 instead of 4–30).
- Not met yet: a second agent vendor.
