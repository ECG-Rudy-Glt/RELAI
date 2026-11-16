# Relai

Relai gives every AI coding agent session, and every person, the same short, generated summary of where a project stands, so nobody starts from zero.

You work with Claude Code, Codex, Cursor or similar agents on one or several repositories. Each new session
re-reads the code, forgets yesterday's decisions, and may rebuild something another of your projects already
has. Relai fixes this with plain files in git and one CLI: no server, no account, nothing but git. It is made
for one developer with several agents first; small teams can use the same files, and an optional
[team server](https://github.com/ECG-Rudy-Glt/RELAI-TEAM) exists for people and agents working in parallel.

## What it looks like

`relai handoff` prints, from files only, a summary of a few hundred tokens (trimmed here):

```text
# Handoff — RELAI
## State
Branch `main` at 0267b05 2026-11-16. Uncommitted files: 0.
Recent commits:
- 0267b05 2026-11-16 Merge pull request #26 from ECG-Rudy-Glt/feat/shared-conventions-hub
## Tasks
Ready (unblocked, 16 total):
- `relai-a7o` P0 Phase 1 — repo contract v0.1 + handoff
## Active changes (specs)
- `shared-conventions-hub` 4/5 tasks (in-progress)
  next: One end-to-end run by hand, recorded in the pull request
## Decisions and pitfalls
- 2026-11-02 — Go, stdlib only — single binary, trivial install, easy to audit.
```

## Install

No Go toolchain needed:

```sh
curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh | sh
```

The installer checks the download against the release's `checksums.txt` and refuses to install on a mismatch
(details and limits in [SECURITY.md](SECURITY.md)). Or download a binary from the
[releases page](https://github.com/ECG-Rudy-Glt/RELAI/releases) (Linux, macOS, Windows; amd64, arm64). From source:

```sh
go install github.com/ECG-Rudy-Glt/RELAI/cmd/relai@latest   # Go ≥ 1.21 fetches the pinned toolchain (1.26.10) by itself
```

## Quick start

```sh
cd your-repo
relai init        # adds AGENTS.md, relai.yaml, .relai/notes.md and, for Claude Code, a session-start hook
relai handoff     # prints where the work stands, in 2,000 tokens or less
relai note add "Use JSON output for integrations"   # records a decision for the next session
```

Nothing is overwritten. Adding Relai to a project that already has docs, skills or a tracker: run `relai discover`.

## For AI agents

- `relai init` writes `AGENTS.md` and, for Claude Code, a hook that loads the handoff at every session start.
- `relai guide` prints a short how-to for any agent.
- `relai guide --install` installs that how-to once as an Agent Skill, so any agent knows the tool in any project.

## Commands

Run `relai <command> -h` for options. `relai tidy`, `relai note archive` and `relai sync` are dry runs unless
`--apply`, and none of them commits.

| Command | What it does |
|---|---|
| **Daily use** | |
| `relai init [dir]` | Add `AGENTS.md`, `relai.yaml`, `.relai/notes.md`, a Claude Code hook; `--hooks` adds git hooks |
| `relai handoff [dir]` | Print the current state of the work; `--json` for tools |
| `relai discover [dir]` | List what the project already gives agents; `--prompt` drafts notes with your agent |
| `relai note add\|check\|archive` | Add a dated line to `.relai/notes.md`, check its size, archive old entries |
| `relai validate [--strict]` | Check `relai.yaml`, `AGENTS.md` size and notes; exit 1 on error (for CI) |
| `relai tidy [--apply]` | After a merge: close tasks a commit says it closes, refresh `.beads/issues.jsonl`, delete merged branches |
| `relai guide [--install]` | Print the how-to for agents, or install it as an Agent Skill |
| `relai version` | Print the version |
| **Several repositories** | |
| `relai index build` | Collect each repo's `relai.yaml` into an index (`--scan <folder>` finds repos) |
| `relai workspace` | One short overview of every indexed project |
| `relai find <capability>` | Which repos already provide it; `--consumers` for who uses it |
| `relai discover --links <folder>` | Mentions between repos, as evidence; `--propose`, `--prompt` |
| `relai new <template> <dir>` | Scaffold a repo from your own template |
| **Shared conventions** | |
| `relai hub init <dir>` | Scaffold a conventions hub |
| `relai sync [--apply]` | Vendor the conventions of the hub pinned in `relai.yaml` |
| `relai check [--outdated]` | `validate` + handoff budget + drift from the hub; for CI |

## Status and limits

Early development (v0.x): everything may change.

- The savings are measured once per case, with real agents, in [`eval/`](eval/README.md): 1.2x to 3.4x in
  dollars, and no gain on repositories with no tracker or notes. Treat them as indications.
- The multi-project overview was measured on 16 repositories of one developer working alone; the team server
  matters when several people or agents work in parallel.
- Not built yet: upstream changes shown in the handoff, shared hooks and per-repo skill selection (see
  [`docs/multi-repo.md`](docs/multi-repo.md)).
- Handoff text is information, not instructions. Do not put secrets in notes, task titles or commit messages.

## Read more

- [`docs/usage.md`](docs/usage.md): the full guide (hub, links, validate, tidy, find, new, other agents, GitHub Issues)
- [`docs/DESIGN.md`](docs/DESIGN.md), [`docs/multi-repo.md`](docs/multi-repo.md), [`docs/HANDOFF.md`](docs/HANDOFF.md)
- [`spec/handoff.md`](spec/handoff.md), [`spec/relai-yaml.md`](spec/relai-yaml.md): the formats
- [SECURITY.md](SECURITY.md), [CONTRIBUTING.md](CONTRIBUTING.md)

## Licence

[Apache-2.0](LICENSE).
