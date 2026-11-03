# Spec Delta

## Purpose

Lets a person or small team see what capability another one of their projects already provides,
so they can reuse it instead of building a duplicate, using nothing but files already committed in
each project's `relai.yaml`.

## ADDED Requirements

### Requirement: Index is built from a flat list of repo paths
The system SHALL build an index from a plain list of local repo paths or clone URLs (a
`repos.txt`-style file, or an explicit list passed on the command line), reading each repo's
`relai.yaml` without requiring push access or any hosted service.

#### Scenario: Building the index from local clones
- **WHEN** the user runs `relai index build` with a file listing paths to repos already cloned
  on disk
- **THEN** the system reads each repo's `relai.yaml`, skips any repo that has none, and writes a
  single index file collecting every readable `relai.yaml` plus the repo path it came from

#### Scenario: A listed repo is missing or has no relai.yaml
- **WHEN** a path in the list does not exist, is not a git repo, or has no `relai.yaml`
- **THEN** the system skips that entry, reports it as skipped with the reason, and still builds
  the index from the remaining repos

### Requirement: Finding existing providers of a capability
The system SHALL let a user search the index for a capability name and list every indexed repo
whose `relai.yaml` declares that capability under `provides:`.

#### Scenario: Capability already provided elsewhere
- **WHEN** the user runs `relai find <capability>` and the index contains a repo whose
  `relai.yaml` lists `provides: [{capability: <capability>}]`
- **THEN** the system prints that repo's id, path or URL, and its `summary:`, so the user can
  decide to reuse it instead of building a duplicate

#### Scenario: No existing provider
- **WHEN** the user runs `relai find <capability>` and no indexed repo provides it
- **THEN** the system states plainly that nothing in the index provides it, without error, so the
  user knows to proceed with new work

#### Scenario: Index does not exist yet
- **WHEN** the user runs `relai find <capability>` before any `relai index build` has run
- **THEN** the system reports that no index was found and names the command to build one, instead
  of failing with an unrelated file-not-found error

### Requirement: Index building is read-only and pull-based
The system SHALL only read from each listed repo when building the index; it SHALL NOT write to,
push to, or require write credentials for any indexed repo.

#### Scenario: Building the index over repos the user cannot push to
- **WHEN** the user has read-only or fetch-only access to a listed repo
- **THEN** `relai index build` still succeeds, since it only reads `relai.yaml` from that repo
