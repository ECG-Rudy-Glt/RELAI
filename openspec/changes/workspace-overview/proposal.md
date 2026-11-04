# Proposal

## Why

`relai handoff` answers "where are we" for one repository. A person or agent working across several
projects has no equivalent: to decide where to go, it opens each repository's handoff, which costs
about 500 tokens each and, through `bd`, rewrites about 3 MiB of the Beads database per project.
The index (`relai index build`) already knows every project; it lacks only a compact view of activity
and open work.

## What Changes

- New command `relai workspace [--index <path>] [--max-tokens N]`: one deterministic overview of all
  indexed projects (default 800 tokens), most recently active first, with open and in-progress task
  counts, the most urgent tasks and what each project provides.
- It reads files only: the index, each project's `.beads/issues.jsonl` copy and the date of its last
  commit. It never runs `bd`.
- Budget is met by showing fewer tasks, then fewer capabilities, then the least active projects by
  name only, then only their number.

## Impact

- New command; nothing existing changes. Task counts may lag the live list (the file is a copy), which
  the document says.
