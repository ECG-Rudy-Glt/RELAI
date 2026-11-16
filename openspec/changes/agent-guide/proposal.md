## Why

An agent can use relai well only where `relai init` wrote AGENTS.md and the session hook. Anywhere else
it has to be told. A built-in how-to, installable once per user, removes the explanation.

## What Changes

- `relai guide` prints a short, vendor-neutral how-to for agents.
- `relai guide --install` writes it as an Agent Skill in `~/.claude/skills/relai/SKILL.md` and
  `~/.agents/skills/relai/SKILL.md`; `--force` overwrites a file that differs.

## Impact

New package `internal/guide`, new file `cmd/relai/guide.go`, README. Closes relai-yfa.
