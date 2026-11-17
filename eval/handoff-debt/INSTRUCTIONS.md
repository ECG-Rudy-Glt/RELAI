# Running the Handoff Debt benchmark with Claude agents — instructions for another machine

**Status: untested.** Written from the public README of the benchmark and the paper, without running it. Where a step
says "check", the README did not say. Do a one-task smoke test (step 4) before anything else, and stop at the first
step that does not work: write down which one, that is already a result.

**Why.** Handoff Debt ([arXiv 2606.02875](https://arxiv.org/abs/2606.02875), code
[anjilab/agent-handoff-debt](https://github.com/anjilab/agent-handoff-debt), MIT) measures the extra effort an agent
spends when it takes over a task another agent interrupted, on 75 SWE-bench Verified tasks, with four ways of handing
over: repository only, raw trace, summary notes, structured notes. Relai's claim is that a compact handoff does at
least as well as their notes at a small token cost. We want a **fifth view: the output of `relai handoff`**, and Claude
as the successor (the paper used Qwen, Gemma and Devstral). Our own small test is `eval/resume` and
`eval/results/2026-11-17-resume.md`.

## What you need

- A Linux machine, **100 GB of free disk** (SWE-bench images are large; the benchmark does not say how large), 16 GB RAM,
  Docker that you may fill, Python 3.12 or 3.13, [uv](https://docs.astral.sh/uv/), git.
- An Anthropic API key (or the Claude access you use), and a budget. The README gives no cost; start with one task and
  read the bill before going on. Our 48 small runs cost about 3 dollars API-equivalent; a SWE-bench task is far bigger.
- `relai` v0.11.0 or later (`curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh | sh`).

## Steps

1. `git clone https://github.com/anjilab/agent-handoff-debt && cd agent-handoff-debt && uv sync`
2. **A Claude endpoint.** The harness talks to an OpenAI-compatible endpoint (`HANDOFF_LLM_BASE_URL`,
   `HANDOFF_LLM_API_KEY`, `HANDOFF_LLM_MODEL`). Claude is not mentioned in the README. Two ways to try, in this order:
   a. Anthropic's OpenAI-compatible endpoint, if your account has it (check Anthropic's current docs for the URL);
   b. a LiteLLM proxy (`pip install 'litellm[proxy]'`, `litellm --model anthropic/<model>`) and point the variables at it.
   Copy `configs/agents/qwen.toml` to `configs/agents/claude.toml` and set the model name. Check that tool calling works:
   `uv run handoff-debt doctor --agent-config configs/agents/claude.toml` then `model-smoke`.
3. Read `src/handoff_debt/` for how the four views are built and injected, and for the checkpoint kinds
   (`after_first_validation_result` is the one the README names). Add a fifth view `relai` that runs
   `relai handoff` in the frozen repository at the checkpoint and uses its output as the handoff text. If the repository
   at the checkpoint has no `AGENTS.md`/`relai.yaml`, add a short one first (the handoff then works in degraded mode,
   which is also what a repository adopting Relai late looks like). Keep that change in a patch file.
4. **Smoke test, one task** (from the README, with the Claude config):
   ```sh
   uv run handoff-debt run-task --task-config configs/tasks/swebench_verified/sphinx-doc__sphinx-8548.toml --agent-config configs/agents/claude.toml --runs-dir data/runs
   uv run handoff-debt finalize-run --run-dir data/runs/sphinx-doc__sphinx-8548/claude/initial
   uv run handoff-debt takeover --initial-run data/runs/sphinx-doc__sphinx-8548/claude/initial --agent-config configs/agents/claude.toml --handoff-view structured_notes --checkpoint-kind after_first_validation_result --runs-dir data/runs
   ```
   Note the time, the disk used by Docker (`docker system df`) and the cost. If the predecessor run is too heavy, the paper's
   predecessor trajectories are on Hugging Face (`kc-academy/handoff-debt`, `load_dataset("kc-academy/handoff-debt", "runs", split="records")`);
   the README does not say whether they can be placed in the `--initial-run` layout: check.
5. If the smoke test works, run **10 to 15 tasks** from `selected_075.txt` (fixed seed: the first ones of the list), for the
   views `repository_only`, `structured_notes` and your new `relai`, one run each, then repeat the best case twice. Keep the
   successor and the predecessor model the same for all.
6. Compare per view: solved rate, agent events, prompt tokens (the harness records events and prompt tokens; the paper's
   Table 2 is the model to follow).

## What to send back

The commit of the benchmark you used, your patch for the `relai` view, the Claude model and the endpoint (direct or
LiteLLM), the smoke-test time/disk/cost, the table of step 6, and every step that failed with the error. Do not edit the
task selection to improve a result. With 10 to 15 tasks and one run each the result is an indication; say so.

## Reading the result

The paper found the note-based views cut events by 20% to 59% and prompt tokens by 42% to 63%, with solved-rate gains that
were small and model-dependent. Relai's handoff is worth keeping if the `relai` view is in the same range at a few
hundred to 2,000 tokens. If it is clearly worse than structured notes, that is a finding about the handoff format, not
about the benchmark.
