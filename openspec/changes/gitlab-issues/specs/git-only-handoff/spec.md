## ADDED Requirements

### Requirement: GitLab Issues as an opt-in task backend
When a repository has no `.beads/` and its `relai.yaml` says `tasks: gitlab`, the handoff SHALL list
the open GitLab issues read with `glab`, once, in at most 8 seconds. Without that line the handoff
SHALL NOT run `glab` nor use the network. Relai SHALL NOT handle any token or URL for it.

#### Scenario: Issues are listed
- **WHEN** `relai.yaml` says `tasks: gitlab` and `glab` returns open issues
- **THEN** the Tasks section lists assigned issues, then unassigned ones with their labels, by `#iid`

#### Scenario: Not asked for
- **WHEN** the origin's host contains `gitlab` but `relai.yaml` has no `tasks: gitlab`
- **THEN** `glab` is not run and the Tasks section says how to turn the adapter on

#### Scenario: glab fails
- **WHEN** `glab` is missing, exits with an error, times out or prints something unreadable
- **THEN** the Tasks section gives a fixed reason and no raw tool output

#### Scenario: Turned off on a machine
- **WHEN** `RELAI_NO_GH` is set
- **THEN** `glab` is not run
