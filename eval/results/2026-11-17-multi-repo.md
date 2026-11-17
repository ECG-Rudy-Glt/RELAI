# Multi-repository measure — 2026-11-17

Fixture `eval/multi/build.sh`: web-app (stale README/TODO says /v1/invoices) links to billing-api, which moved to /v2 after web-app's last commit; two decoys. Run: `eval/multi/run.sh <model> <none|docs|relai> <n> <out.jsonl>`. Claude Code headless, read-only tools, 3 runs per cell.

| model | condition | correct | turns | input tokens | cost USD |
|---|---|---|---|---|---|
| opus | docs | 3/3 | 12 | 156873 | 0.13 |
| opus | none | 3/3 | 9 | 173551 | 0.23 |
| opus | relai | 3/3 | 10 | 141612 | 0.13 |
| sonnet | docs | 3/3 | 9 | 164997 | 0.07 |
| sonnet | none | 3/3 | 7 | 162307 | 0.07 |
| sonnet | relai | 3/3 | 6 | 103204 | 0.06 |

All 18 runs found the right step: the fixture is easy, agents also explore sibling repositories without help. The gain is tokens and turns (Sonnet 103k vs 162-165k), not correctness. One vendor (Claude), n=3: indications. No second vendor (Codex, Gemini...) is installed on this machine, so relai-a7o.10 stays open.

## Hard variant (`VARIANT=hard`)

billing-api's README is stale too (still /v1); the change is only in a commit message and a closed task; six busy decoy repos (a "v2 parser" each). 3 runs per cell.

| model | condition | correct | turns | input tokens | cost USD |
|---|---|---|---|---|---|
| haiku | docs | 0/3 | 14 | 368323 | 0.1 |
| haiku | none | 0/3 | 11 | 258083 | 0.09 |
| haiku | relai | 0/3 | 5 | 129953 | 0.03 |
| opus | docs | 3/3 | 13 | 203189 | 0.15 |
| opus | none | 3/3 | 8 | 175331 | 0.14 |
| opus | relai | 3/3 | 8 | 134192 | 0.13 |
| sonnet | docs | 3/3 | 10 | 184020 | 0.08 |
| sonnet | none | 3/3 | 8 | 171880 | 0.07 |
| sonnet | relai | 3/3 | 8 | 123407 | 0.07 |

Sonnet and Opus found the right step in all 18 runs again, with Relai tokens down 28% (Sonnet 123k vs 172-184k) and 23-34% (Opus 134k vs 175-203k). Haiku fell for the stale /v1 in all 9 runs, **including with the workspace handoff**: it read the upstream-changes line and still did not look at billing-api (5 turns, 130k tokens: cheaper, but wrong). So the upstream line is a signal a capable model follows and a small one does not; wording it as a stronger instruction was deliberately not tried on this same fixture, to avoid tuning to the test.

## Second vendor: Codex (run by a colleague, 2026-11-17)

`eval/handoff-debt/rapport.md` and `results-codex.jsonl` (the raw evidence archive is kept out of the repository). Codex CLI 0.169.0, model `gpt-6.1-sol`, Windows, 3 runs per cell, the same fixtures and grading (adapted launcher, PowerShell). Correct answers: easy none/docs/relai 3/3 each; hard none 2/3, docs 2/3, **relai 1/3**. Input tokens: easy 65.6k / 81.1k / 76.0k, hard 89.3k / 111.0k / 79.0k (none / docs / relai). So with a second vendor the result does not confirm an advantage of the handoff: on the hard variant it was not better than no help, and in the two failed `relai` runs Codex read the line saying `billing-api` had moved and still wired `/v1`, as Haiku did. Three runs per cell is not enough to separate 1/3 from 2/3. What it does support: the weak point is the upstream line itself, which said *that* a neighbour moved and not *what* changed; format v0.10 adds the subjects of the new commits. That change has not been re-measured, and the hard fixture is now too easy to test it (the answer is in the line): a different fixture is needed.
