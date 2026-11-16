# Proposal

## Why

After work is merged, three chores wait for someone to remember them: closing the tasks the work finished,
refreshing the committed task export, and deleting merged branches. They were left undone on 2026-11-04 (about thirty
tasks still "in progress" after their code was merged). They are mechanical and should happen by themselves.

An independent test the same day also found three small defects (a repeated `provides:` key lost a list, the pre-commit
hook had no timeout) and the discovery needed its first follow-ups (nested folders, use of `links:`, a better rule than a
guessed file count for what to propose).

## What Changes

- `relai tidy [--apply]`: closes the tasks that a commit on the default branch says it closes ("Closes <id>"), refreshes
  `.beads/issues.jsonl` (owner field removed), deletes fully merged local branches. Dry run by default, never commits,
  never closes from a feature branch, never closes a task that is only named.
- `relai init --hooks` also installs a `post-merge` hook that runs it after every pull or merge; the `pre-commit` hook now
  gives `bd` at most 20 seconds.
- A repeated `provides:`/`consumes:` key keeps both lists and warns.
- `relai discover --links --depth N` and `relai index build --scan --depth N` look inside folders; nested repos are named by path.
- `relai find` shows `works with:` and puts first the repos linked with the one you are in (`[linked from here]`).
- `--propose` proposes a repo mentioned in two or more CI/deploy/config files, or in `--min-weak` files of any kind (default 5).

## Impact

- New command and hook; additive flags. Nothing is closed without an explicit keyword and an id that is open.
