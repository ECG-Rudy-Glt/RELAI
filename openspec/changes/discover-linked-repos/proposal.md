# Proposal

## Why

The goal is to drive several linked repositories from one place. Today nothing finds them: the index needs a
hand-written list, `provides`/`consumes` are typed by hand, and a repository that never ran `relai init` is invisible.
A person who has twenty projects in one folder gets nothing until they have done all of that.

## What Changes

- `relai index build --scan <folder>`: finds the git repositories of a folder (symbolic links to repositories included),
  indexes them, and says which have no `relai.yaml` yet.
- `relai discover --links <folder>`: the MENTIONS between those repositories. Strong evidence: the other repository's
  remote URL or a relative path to it. Weak evidence: its name as a word (names under 4 letters are not searched).
  Each link carries counts and up to three file paths as evidence, never content.
- `--propose` prints a `links:` block per `relai.yaml` for a person to check and paste; `--prompt` prints instructions for an
  agent to verify each mention with the engineer and write only after they agree; `--json` for tools.
- `relai.yaml` gets an optional `links:` key (ids of the repositories this one works with), read by the index and shown
  by `relai workspace`.

## Impact

- New flags and one optional manifest key. Nothing is ever written by the discovery itself.
