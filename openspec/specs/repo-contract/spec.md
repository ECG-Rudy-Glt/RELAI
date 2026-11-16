# repo-contract Specification

## Purpose
TBD - created by archiving change repo-contract-validate. Update Purpose after archive.

## Requirements

### Requirement: relai.yaml has a versioned one-page specification
The format of `relai.yaml` SHALL be specified in `spec/relai-yaml.md` with a version field
(`apiVersion`) and a compatibility rule, and every reader in this repository SHALL give the result
recorded in `spec/conformance/` for each case.

#### Scenario: Two readers agree
- **WHEN** `go test ./internal/index` runs with `python3` available
- **THEN** the Go reader and `spec/readers/relai_yaml.py` both return the expected model for every conformance case

#### Scenario: Unknown keys are ignored
- **WHEN** a `relai.yaml` contains a top-level key the format does not define
- **THEN** readers skip it and still return the known keys

### Requirement: relai validate checks the repo contract
`relai validate` SHALL report, for the repository it is given, every departure from the contract as
an error or a warning, and SHALL exit with code 1 when there is an error, or any finding with `--strict`.

#### Scenario: A repository that follows the contract
- **WHEN** `relai.yaml` has `apiVersion`, `id` and `summary`, `AGENTS.md` is at most 100 lines and `.relai/notes.md` exists
- **THEN** `relai validate --strict` prints `ok` and exits 0

#### Scenario: A required key is missing
- **WHEN** `relai.yaml` has no `id` or an empty `summary`
- **THEN** an error names the key and the command exits 1

#### Scenario: A mistyped key
- **WHEN** `relai.yaml` contains `provide: [a]`
- **THEN** a warning says the key is unknown, the command exits 0, and exits 1 with `--strict`

#### Scenario: A value that was not understood
- **WHEN** a `provides` entry has no `capability:` key
- **THEN** it is reported as an error, never dropped silently

#### Scenario: AGENTS.md over its budget
- **WHEN** `AGENTS.md` has more than 100 lines
- **THEN** an error gives the line count and the budget

#### Scenario: A newer format version
- **WHEN** `apiVersion` is not `relai.dev/v0`
- **THEN** an error says this relai does not read that version
