# Your AI agent re-reads your whole repo every session. It doesn't have to.

*Draft for review — not yet published. Adapts [`docs/DESIGN.md` §11](../DESIGN.md#11-market-scan)
(market scan) and [`eval/README.md`](../../eval/README.md) (E2 results). The E2 numbers can be
re-run with `go run ./eval/e2` (needs a logged-in `claude`, uses quota); `-dry` only builds the
fixtures and prints the prompts. All agents tested so far are Claude models.*

## The problem

Every time an AI coding agent starts a session cold, it re-derives the same thing a teammate would
ask you in the first five minutes: where is this project, what's the next step, what already
broke once. Today that means reading `README.md`, `TODO.md`, the journal, maybe the whole git log
— thousands to tens of thousands of tokens, every single time, with no guarantee the agent lands
on the actually-current state instead of a stale TODO.

We first estimated this on a real ~420-commit infrastructure repo (51 Beads tasks, 2 OpenSpec
changes) from the *sizes* of the files a cold agent would read (~96,000 tokens) against the size of
`AGENTS.md` plus a generated handoff (~1,700). That is an estimate of what would be read, not a
measurement of an agent, and an agent does not read everything: when we ran real agents the gain
was smaller (below), so treat the estimate as an upper bound on what there is to save.

## What we built

[Relai](https://github.com/ECG-Rudy-Glt/RELAI) is a small, vendor-neutral layer — files committed
to git, plus one Go binary, no server — that gives any AI agent (Claude Code, Codex, Copilot,
whatever's next) a compact, deterministic answer to "where are we and what's next," generated from
the project's existing git history, task tracker (Beads) and specs (OpenSpec) instead of written
by an LLM:

```sh
curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh | sh
cd your-repo && relai init && relai handoff
```

`relai handoff` reads git, Beads and OpenSpec state and renders a ≤2,000-token Markdown report:
branch and uncommitted files, open tasks (flagging ones that look done but were never closed —
that cost us a wrong answer in testing, see below), the next unchecked step of any active spec
change, and whatever the team already uses (skills, hooks, CI) by way of `relai discover`. No
tracker, no specs, not even `AGENTS.md`? It falls back to a **degraded mode**: a git-only report
(recent commits, recently touched files, `TODO`/`FIXME` markers) instead of nothing — Relai's
value shouldn't require adopting the rest of the stack first.

It also ships `relai find <capability>`: before you build something, check whether one of your
other projects already provides it, by searching a flat index of their `relai.yaml` files. No
portal, no account — just `relai index build` over a list of repo paths or clone URLs, then
`relai find`.

## Does it actually work? (not just "is it short")

A short answer that's *wrong* is worse than no answer — a compact handoff an agent trusts
blindly is a real failure mode, not a hypothetical one. So before shipping the format we built an
eval harness ([`eval/`](../../eval/README.md)) with four fixture repos whose correct "what's next"
answer is known in advance, including two deliberately adversarial ones: a `TODO.md` that names a
task already merged, and a task tracker where a task was finished but never closed (so the
straightforward "ready" list points at the wrong thing). We ran Claude Haiku and Sonnet, headless,
read-only, three conditions: the prompt alone, "read the docs first," and Relai's handoff.

| | no context | read the docs | Relai handoff |
|---|---|---|---|
| Correct next step, every fixture, both models | yes | yes | yes |
| Tokens vs "read the docs" (raw input tokens) | — | baseline | **1.6×–16× fewer**; ≥ 10× in 2 of 8 cells |
| Cost in dollars, same runs | — | baseline | 1.2×–3.4× less (about $0.02–0.08 per session) |
| Turns to the answer | 4–30 | 4–25 | 1–5 |

The handoff only matched "read the docs" on correctness after we added two things the eval forced
on us: flagging tasks that are still open but named in a recent commit subject (probably done,
nobody closed it), and showing blocked tasks instead of only "ready" ones. Without those, a stale
task list made Haiku confidently wrong in one turn where reading the full docs — 16× more tokens —
got it right. The fix cost nothing in tokens; it's in the format now.

On four of our own real repos (not fixtures, one run each): the one with a task tracker and specs
gave the same answer as reading the docs for about **3× fewer tokens** (30k against 101k) — that is
a single repo, not a trend. On the one with no tracker and no notes the handoff cost *more* (113k
against 94k). It is barely more than `git log` there and saves nothing — which is exactly why degraded mode
exists, and exactly why we're honest that the ceiling is bounded by what the repo already tracks,
not by Relai.

## What's actually new here

Everything *around* this already has standards: `AGENTS.md` is hosted by the Agentic AI Foundation,
Agent Skills are an open standard, OpenSpec and Beads already do specs and tasks well. We don't
replace any of that — `relai init` and `relai discover` are built to adopt into whatever a repo
already has, not move it.

What's missing, as far as we could tell from a market scan of Anthropic, OpenAI, GitHub/Microsoft,
Google and the developer-portal space (Backstage, Port, OpsLevel): a **standard, vendor-neutral
handoff format with a token budget**, and **cross-repo "check before you build" without a portal**
— every anti-duplication story we found assumes a server and a platform team. Full writeup:
[`docs/DESIGN.md` §11](../DESIGN.md#11-market-scan).

## Try it / poke holes in it

- Repo: [github.com/ECG-Rudy-Glt/RELAI](https://github.com/ECG-Rudy-Glt/RELAI)
- Eval harness (reproduce the numbers above, or run it against your own repos):
  [`eval/README.md`](../../eval/README.md)
- Format spec: [`spec/handoff.md`](../../spec/handoff.md)

It's early (v0.x). We'd rather hear "this breaks on X" now than after we publish the format as a
spec. The one thing we're confident of: whatever the right design turns out to be, it should stay
possible to adopt with nothing but git.
