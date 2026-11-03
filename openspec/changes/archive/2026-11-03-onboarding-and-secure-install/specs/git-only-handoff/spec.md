## ADDED Requirements

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
