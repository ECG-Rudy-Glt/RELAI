# git-only-handoff Specification

## Purpose
Lets `relai handoff` give a useful report in any git repository, even one that has adopted none
of `AGENTS.md`, OpenSpec, or Beads, so Relai's value does not depend on a team first adopting the
rest of the composed ecosystem.

## Requirements

### Requirement: Degraded mode activates when no harness is detected
The system SHALL run `relai handoff` in degraded mode when `AGENTS.md`/`CLAUDE.md`, an OpenSpec
`openspec/` directory, and a Beads `.beads/` directory are all absent, instead of producing an
empty or near-empty report.

#### Scenario: Bare repo with none of the composed tools
- **WHEN** `relai handoff` runs in a git repository that has no `AGENTS.md`, no `CLAUDE.md`, no
  `openspec/`, and no `.beads/`
- **THEN** the system produces a degraded-mode report instead of an empty one, and labels it as
  degraded mode so the reader knows some sections are unavailable

### Requirement: Degraded mode reports git state without other sources
In degraded mode, the system SHALL report branch, latest commit, uncommitted file count, and
recent commit subjects, using only `git`, the same as in full mode.

#### Scenario: Git section is unchanged by degraded mode
- **WHEN** `relai handoff` runs in degraded mode
- **THEN** the branch, latest commit, uncommitted-file count, and recent commits sections are
  identical in content and format to what full mode would print for the same repository state

### Requirement: Degraded mode surfaces open work from the working tree
In degraded mode, the system SHALL list files changed in the last 7 days and occurrences of
`TODO`/`FIXME` markers found by `git grep` in tracked files, bounded by the same token budget as
full mode.

#### Scenario: Recently touched files and open markers
- **WHEN** `relai handoff` runs in degraded mode on a repo with files modified in the last 7 days
  and tracked files containing `TODO`/`FIXME` markers
- **THEN** the report lists the recently touched file paths and the `TODO`/`FIXME` matches (file
  and line), truncating the longer of the two lists first if the token budget would otherwise be
  exceeded

#### Scenario: Nothing recent and no markers
- **WHEN** `relai handoff` runs in degraded mode on a repo with no files touched in the last 7
  days and no `TODO`/`FIXME` matches
- **THEN** the report states plainly that no recent activity or open markers were found, rather
  than omitting the section silently

### Requirement: Degraded mode never fabricates tasks or decisions
The system SHALL NOT infer, summarize, or guess at tasks, decisions, or pitfalls in degraded mode;
it SHALL only report what it can read directly from git and the working tree.

#### Scenario: No task tracker present
- **WHEN** `relai handoff` runs in degraded mode and there is no Beads or OpenSpec data to read
- **THEN** the report omits the Tasks and Active changes sections entirely rather than inventing
  placeholder content for them

### Requirement: Degraded mode stays within the handoff token budget
The system SHALL enforce the same token budget on degraded-mode output as on full-mode output, by
truncating lists, never by rewording or summarizing with an LLM.

#### Scenario: Degraded report exceeds budget before truncation
- **WHEN** the recently-touched-files and `TODO`/`FIXME` lists would together exceed the handoff
  token budget
- **THEN** the system truncates the lists the same way full mode truncates task and commit lists,
  and states how many entries were omitted

### Requirement: The handoff says when tasks cannot be read
When a repo has a task backend but its tasks cannot be read in the current environment, the handoff
SHALL say so with a short reason and how to load them, SHALL NOT present an empty list of ready tasks,
and SHALL NOT include raw output of the failing tool. If a snapshot of the tasks is kept in git, it SHALL
show the unfinished tasks from it, labelled as a snapshot that may lag.

#### Scenario: Fresh clone without a Beads database
- **WHEN** `.beads/` exists, `bd` is installed, and it reports no database
- **THEN** the handoff says the clone has no database yet, tells how to create it, and lists the
  unfinished tasks from `.beads/issues.jsonl` without any "Ready: 0" line

#### Scenario: bd is not installed
- **WHEN** `.beads/` exists and the `bd` command is missing
- **THEN** the handoff says `bd` is not installed and still shows the snapshot

#### Scenario: bd fails for another reason
- **WHEN** `bd` fails with an unrecognised error
- **THEN** the handoff says it failed to read the tasks and shows none of the tool's output

#### Scenario: Tasks are readable
- **WHEN** `bd` answers normally
- **THEN** the live tasks are shown and the snapshot and the warning are not
