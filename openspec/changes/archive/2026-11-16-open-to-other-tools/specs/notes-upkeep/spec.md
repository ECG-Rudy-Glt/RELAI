## ADDED Requirements

### Requirement: Notes are kept short with whole-line operations
`relai note` SHALL add dated entries to `.relai/notes.md`, list the dated entries older than a given
number of days, and move them to `.relai/notes-archive.md` only on `--apply`, never rewriting or
dropping a line.

#### Scenario: Adding a decision
- **WHEN** `relai note add "text"` runs
- **THEN** `- <today> — text` is appended at the end of the Decisions section, created if missing

#### Scenario: Checking
- **WHEN** `relai note check` runs
- **THEN** it prints the size of the notes against the handoff budget and the entries older than 90 days

#### Scenario: Archiving is a dry run by default
- **WHEN** `relai note archive` runs without `--apply`
- **THEN** no file changes

#### Scenario: Archiving loses nothing
- **WHEN** `relai note archive --apply` runs
- **THEN** every old entry, with its continuation lines, is in the archive file and no longer in the notes, and undated lines are untouched
