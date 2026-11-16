## ADDED Requirements

### Requirement: A repository vendors shared conventions from a pinned hub
A repository that pins a hub with `conventions: <hub>@<tag>` in `relai.yaml` SHALL get the hub's
conventions and skills as plain files through `relai sync`, which SHALL change nothing without
`--apply` and SHALL touch only the content it manages.

#### Scenario: Dry run
- **WHEN** `relai sync` runs without `--apply`
- **THEN** it prints what would change and no file is modified

#### Scenario: Local rules survive
- **WHEN** `relai sync --apply` runs on a repository whose `AGENTS.md` has local rules
- **THEN** only the managed block and the vendored files change

### Requirement: Drift from the hub is detected
`relai check` SHALL fail when vendored content differs from the hub at the pinned tag, and SHALL
list declared overrides without failing.

#### Scenario: A hand edit
- **WHEN** `.relai/conventions.md` was edited by hand
- **THEN** `relai check` reports an error and exits 1

#### Scenario: A declared override
- **WHEN** `AGENTS.md` has a local line starting with `Override:`
- **THEN** `relai check` lists it and still exits 0

### Requirement: Hub content is untrusted
`relai sync` SHALL write only inside the repository and only to the files it manages, and SHALL
skip and report anything in the hub that is unsafe.

#### Scenario: A hostile hub
- **WHEN** the hub has a skill named with `..`, a symbolic link, or a file over 1 MiB
- **THEN** it is skipped and reported, and nothing is written outside the repository

#### Scenario: A value that git would read as an option
- **WHEN** the hub or the tag in `conventions:` starts with `-`
- **THEN** it is refused before git is called

### Requirement: Shared skills follow the hub
`relai sync --apply` SHALL install each hub skill as `hub-<name>` in `.claude/skills/` and
`.agents/skills/`, and SHALL remove a `hub-*` folder the hub no longer has, only when it is a real
folder of this repository.

#### Scenario: A skill removed at a newer tag
- **WHEN** the pin moves to a tag where the hub dropped a skill and `relai sync --apply` runs
- **THEN** that skill's `hub-*` folders are removed and local skills are untouched

### Requirement: `relai check` combines the contract and the drift
`relai check` SHALL report the findings of `relai validate`, a handoff over its budget, and drift
from the hub at the pinned tag; it SHALL exit 1 on an error, and on a warning with `--strict`, and
never on an "info" finding.

#### Scenario: A newer hub version
- **WHEN** `relai check --outdated` runs and the hub has a higher version tag (compared numerically)
- **THEN** it prints a warning and exits 0, or 1 with `--strict`
