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
