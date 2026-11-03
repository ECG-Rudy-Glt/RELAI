# Design

## Decisions

**Say "cannot read" instead of showing an empty list.** An empty "Ready" list is indistinguishable from
"nothing to do", and that is exactly what an agent in a fresh clone concluded. The handoff now tells
three cases apart (no task backend, backend present and readable, backend present but unreadable) and in
the last one names the fix. The reason is picked from a short fixed set; raw tool output is never
copied, because it could carry paths or secrets into an agent's context.

**Fall back to the snapshot in git, labelled.** `.beads/issues.jsonl` is the copy of the task list that
travels with the repo. Showing its unfinished tasks gives a clone something useful before `bd bootstrap`,
and also on a machine without `bd`. It is labelled as a snapshot that may lag so nobody trusts it as live.

**`CLAUDE.md` imports `AGENTS.md` instead of copying it.** One source of truth; and it works on every Claude
Code version, whereas reading `AGENTS.md` directly depends on the version. If a `CLAUDE.md` already exists
it is left alone and `relai init` says what to add.

**A SessionStart hook, because instructions are advisory.** The hook script's output is added to the
session context, so loading the handoff stops depending on the agent remembering to. It lives in a script
(testable, reviewable) that `settings.json` points to. It never exits non-zero and falls back to the plain
files when `relai` is not installed, so it cannot block a session on a machine without the tool.

**Hooks are code from the repo.** The README and SECURITY.md say so: Claude Code asks for trust before
running a project's hooks, and people should read hooks of repos they do not know. `relai init` never
modifies an existing `settings.json`.

**Checksum, honestly named.** The installer compares the archive's SHA-256 with `checksums.txt`, fails
closed (no checksum file, no tool to compute it, no entry, mismatch), and has an explicit opt-out that
warns. Because both files come from the same release it does not defend against a compromised release;
the docs say that and point to building from source.

**The task export drops the owner field**, which is the creator's e-mail address, so publishing the task
list does not publish personal data.

## Risks / Trade-offs

- The snapshot can be stale. Mitigated by the label and by `scripts/sync-beads.sh` in the contribution rules.
- A hook adds a `relai handoff` run (three `bd` calls, ~9 MiB of disk writes) per session start. Fine per
  session; reducing it is tracked separately.
