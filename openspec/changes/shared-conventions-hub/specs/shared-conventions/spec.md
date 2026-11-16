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
