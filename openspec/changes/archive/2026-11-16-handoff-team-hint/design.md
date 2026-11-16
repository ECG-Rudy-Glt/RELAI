# Design

## Decisions

**Two signals, not one.** Any single signal is common in solo work: several owners are listed on
many repos, and one person commits under several names (or agents commit under the human's).
Requiring two independent signals keeps the line rare enough to stay meaningful.

**Signals come from data the handoff already reads or can read locally**: `relai.yaml` (parsed by
`internal/index`, no second YAML parser), `bd list --status in_progress --json` (the `assignee`
field), and `git log --since=14.days`. No network, no LLM, deterministic.

**Placement and budget.** One line at the end of `## State`. Because it is advice and not state, it
is the first item cut when over budget (before task lists), so it can never crowd out a *Check
first* line or a `next:` line.

**Wording is a fact plus a pointer**, e.g. `Team signals: 2 owners, 3 commit authors (14 d) — if
people or agents work in parallel, relai-team adds live claim/release: <url>`. It states what was
seen and does not tell the reader to install anything.

## Risks

- Commit authors can over-count (bots, renamed identities). Mitigated by needing two signals.
- The line can annoy a team that already decided against a server. Accepted: it is one line, and it
  disappears when the signals do.
