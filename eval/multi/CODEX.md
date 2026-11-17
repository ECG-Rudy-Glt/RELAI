# Measuring Relai with Codex — instructions

**Goal.** Repeat the multi-repository measure with a second agent vendor (task `relai-a7o.10`). So far
only Claude (Haiku, Sonnet, Opus) was measured. If Codex also needs fewer tokens or turns with Relai, the
claim holds for two vendors; if not, we need to know.

**What it does.** `build.sh` makes four to ten small git repositories in a temporary folder. `web-app`
links to `billing-api`, which changed its API (`/v1/invoices` to `/v2/invoices`) after web-app's last commit.
The agent must say the next step for web-app; the right answer names `/v2` or the cursor pagination. Three
conditions: `none` (just the question), `docs` (told to read the docs), `relai` (given the output of
`relai handoff --workspace .`). Each run is graded by `run-codex.sh` and appended to a JSONL file.

## Steps (about 20 minutes, 18 runs)

1. Install and log in to Codex (`codex`), and install `jq`, `git` and the `relai` binary
   (`curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh | sh`).
   You need `relai` from the branch that has `handoff --workspace` (PR #31) until it is released:
   `git clone https://github.com/ECG-Rudy-Glt/RELAI && cd RELAI && git checkout feat/workspace-handoff && go build -o /tmp/relai ./cmd/relai`
   (Go 1.23+; set `RELAI_BIN=/tmp/relai`).
2. Check that the script's Codex flags exist: `codex exec --help` must list `--sandbox`, `--json`, `-o`
   (output last message) and `--skip-git-repo-check`. If a name differs, edit `eval/multi/run-codex.sh`.
   The script was written without Codex installed: **do one run first** and read the output line.
3. One pilot run: `RELAI_BIN=/tmp/relai sh eval/multi/run-codex.sh relai 0 /tmp/pilot.jsonl; cat /tmp/pilot.jsonl`.
   `next` must contain a `NEXT:` line. If it is empty, look at the agent's answer and fix the script.
4. The full measure, easy then hard (3 runs per condition each; keep the model name the same for all):
   ```sh
   export RELAI_BIN=/tmp/relai CODEX_MODEL=<the model you use>
   for v in easy hard; do for c in none docs relai; do for n in 1 2 3; do
     VARIANT=$v sh eval/multi/run-codex.sh $c $n results-codex.jsonl
   done; done; done
   ```
5. Summarise:
   ```sh
   jq -rs 'group_by(.variant+.cond)|.[]|"\(.[0].variant) \(.[0].cond): \(map(select(.correct))|length)/\(length) correct, commands \(((map(.commands)|add)/length)|floor), input \(((map(.input)|add)/length)|floor)"' results-codex.jsonl
   ```
6. Send back `results-codex.jsonl`, the summary, the Codex version (`codex --version`), the model, and whether you
   changed the script. Do not edit the fixture or the grading to make a result look better.

## Notes

- Token counts and "commands" are read from Codex's JSON events; the field names may differ between
  versions, so compare a couple of runs by eye before trusting the averages. Compare conditions only with
  each other, not with the Claude numbers (different tokenisers and overheads).
- Costs: usually a few dollars of usage for the 18 runs; run the pilot first.
- Honest reading: on the easy variant every Claude agent found the answer without Relai, so a correctness
  gain is not expected there; look at tokens, commands, and the `hard` variant.
