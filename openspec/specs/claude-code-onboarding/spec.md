# claude-code-onboarding Specification

## Purpose
Makes Claude Code start from the project's current state when someone opens a repo prepared with Relai,
instead of depending on the agent remembering to ask for it.

## Requirements

### Requirement: Init wires Claude Code without overwriting
`relai init` SHALL add a `CLAUDE.md` that imports `AGENTS.md`, a session-start script and the settings
that register it as a SessionStart hook, SHALL never overwrite or edit a file that already exists, SHALL
tell the user what to add by hand when it kept one, and SHALL allow opting out.

#### Scenario: New project
- **WHEN** `relai init` runs in a repo with none of these files
- **THEN** `CLAUDE.md` contains the import line, the hook script is executable, and `settings.json` is
  valid JSON registering a SessionStart hook that runs it

#### Scenario: Existing files
- **WHEN** a `CLAUDE.md` or `.claude/settings.json` already exists
- **THEN** it is left byte-for-byte unchanged and the output says what to add manually

#### Scenario: Opting out
- **WHEN** `relai init --no-claude` runs
- **THEN** no Claude Code file is created

#### Scenario: Running twice
- **WHEN** `relai init` runs a second time
- **THEN** it creates nothing

### Requirement: The session hook never blocks a session
The session-start script SHALL print the handoff when `relai` is installed and otherwise the plain
project files, and SHALL exit successfully in every case, including when the project directory is
unavailable.

#### Scenario: relai is not installed
- **WHEN** the hook runs on a machine without the `relai` command
- **THEN** it prints the contents of `.relai/notes.md` and `relai.yaml` and exits 0

#### Scenario: Broken environment
- **WHEN** the project directory does not exist
- **THEN** it exits 0 and prints nothing harmful
