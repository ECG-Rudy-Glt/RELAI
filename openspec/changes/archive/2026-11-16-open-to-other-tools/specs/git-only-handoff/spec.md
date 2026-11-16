## ADDED Requirements

### Requirement: GitHub Issues as an opt-in task backend
When a repository has no `.beads/` and its `relai.yaml` says `tasks: github`, the handoff SHALL list
the open GitHub issues read with `gh`. Without that line the handoff SHALL NOT run `gh` nor use the
network.

#### Scenario: Issues are listed
- **WHEN** `relai.yaml` says `tasks: github` and `gh` returns open issues
- **THEN** the Tasks section lists assigned issues, then unassigned ones with their labels, by number

#### Scenario: Not asked for
- **WHEN** the origin is on GitHub but `relai.yaml` has no `tasks: github`
- **THEN** `gh` is not run and the Tasks section says how to turn the adapter on

#### Scenario: gh fails
- **WHEN** `gh` is missing or exits with an error
- **THEN** the Tasks section gives a fixed reason and no raw tool output

#### Scenario: Turned off on a machine
- **WHEN** the environment variable `RELAI_NO_GH` is set
- **THEN** `gh` is not run

### Requirement: The handoff is available as JSON
`relai handoff --json` SHALL print the facts of the handoff as JSON, with empty lists as `[]`, and
`relai workspace --json` SHALL do the same for the workspace overview.

#### Scenario: A tool reads the tasks
- **WHEN** `relai handoff --json` runs in a repository with tasks
- **THEN** the output is valid JSON with `format`, `state`, `tasks.source` and the task lists
