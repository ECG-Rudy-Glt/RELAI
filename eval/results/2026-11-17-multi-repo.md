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
