## ADDED Requirements

### Requirement: relai explains itself to any agent
`relai guide` SHALL print a short how-to, identical for every agent, and `relai guide --install`
SHALL write it as an Agent Skill under the user's home directory.

#### Scenario: Print
- **WHEN** `relai guide` runs
- **THEN** the how-to is printed to standard output and nothing is written

#### Scenario: Install is idempotent
- **WHEN** `relai guide --install` runs twice
- **THEN** the second run reports each file as up to date and exits 0

#### Scenario: A differing file is not overwritten
- **WHEN** a target exists with other content and `--force` is not given
- **THEN** no target is written, the differing files are named, and the exit code is non-zero

#### Scenario: Symbolic links are never followed
- **WHEN** a target file or its `relai` directory is a symbolic link
- **THEN** the install is refused, even with `--force`
