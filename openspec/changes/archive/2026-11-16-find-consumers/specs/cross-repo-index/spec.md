## ADDED Requirements

### Requirement: Find lists the consumers of a capability
`relai find --consumers <capability>` SHALL list every indexed repository whose `relai.yaml` declares
the capability under `consumes:`, matching the capability name exactly and case-insensitively, as
`relai find` does for `provides:`. When none does, it SHALL say that nothing in the index consumes it
and exit with status 0. With `--json`, `relai find` SHALL print an object with `capability`, `mode`
(`providers` or `consumers`) and `matches` (an empty array when nothing matches).

#### Scenario: A repository consumes the capability
- **WHEN** the index holds a repository declaring `consumes: auth.oidc` and the user runs `relai find --consumers auth.oidc`
- **THEN** that repository is listed with its id, source and summary, and repositories that only provide it are not

#### Scenario: Nothing consumes the capability
- **WHEN** no indexed repository declares the capability under `consumes:`
- **THEN** the output says nothing in the index consumes it, and the exit status is 0

#### Scenario: JSON output
- **WHEN** the user runs `relai find --consumers --json auth.oidc` and nothing matches
- **THEN** the output is a JSON object with `"mode": "consumers"` and `"matches": []`
