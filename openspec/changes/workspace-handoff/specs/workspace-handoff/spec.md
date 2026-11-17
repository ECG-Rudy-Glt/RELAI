## ADDED Requirements

### Requirement: Workspace handoff
`relai handoff --workspace <folder>` SHALL print one block per git repository found in the folder, from
files and git only, SHALL NOT run `bd` or `openspec`, and SHALL stay within the token budget by showing less
per block, then naming the least recently active repositories only.

#### Scenario: Counts are honest
- **WHEN** a repository has unfinished epics or a committed task list older than its last commit
- **THEN** its block says how many epics are not counted and the date of the list, flagged when older

#### Scenario: Upstream moved
- **WHEN** a `links:` entry names a repository of the same folder with commits after this one's last commit
- **THEN** the block, and the single-repository handoff, say how many commits and the date of the latest
- **AND** a link that matches no repository of the folder is not reported
