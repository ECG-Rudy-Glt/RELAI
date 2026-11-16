## ADDED Requirements

### Requirement: One overview of all indexed projects
The system SHALL print, for an index of projects, one short document listing each project with its last
activity, its open and in-progress task counts, its most urgent tasks and what it provides, within a token
budget, ordered by most recent activity first, identically for the same inputs whatever the order of
the index.

#### Scenario: Several projects
- **WHEN** the user runs `relai workspace` with an index of several local projects
- **THEN** the most recently active project comes first and each shows its counts, top tasks and capabilities

#### Scenario: Too many projects for the budget
- **WHEN** the overview does not fit the budget
- **THEN** detail is dropped first, then the least active projects are listed by name only, and the number of projects is always shown

### Requirement: The overview never runs the task tool
The system SHALL build the overview from files only (the index, each project's committed task export, the date of its
last commit) and SHALL NOT run `bd` or any other tool that writes to a project's data.

#### Scenario: Task tool present
- **WHEN** `bd` is installed and projects use Beads
- **THEN** the overview is built without running it
