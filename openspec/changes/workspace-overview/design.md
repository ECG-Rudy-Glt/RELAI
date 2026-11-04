# Design

- **Files, not `bd`.** Each `bd` call rewrites its database; an overview over N projects must not cost
  N database rewrites. `.beads/issues.jsonl` is already committed (via `scripts/sync-beads.sh`).
  The price is possible lag, stated in the document.
- **Order by last commit, ties by id.** Deterministic whatever the order of the index.
- **Remote (URL) entries are listed, not read.** Cloning every project would be slow and needs
  credentials; `relai index build` can be run from a machine that has the clones.
- **Same budget idea as the handoff.** Fewer items, never rewording.
