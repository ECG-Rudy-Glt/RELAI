# Spec Delta

## Purpose

Lets a person or team start a new repository pre-wired with the Relai repo contract from a
reusable template — their own, not one Relai ships — in one command, instead of assembling
`AGENTS.md`, `relai.yaml` and CI/deployment config by hand every time.

## ADDED Requirements

### Requirement: Scaffolding a new repo from a template
The system SHALL create a new directory from a template (a local path or a clone URL) and the
Relai repo contract, given an explicit template source and a target directory.

#### Scenario: Scaffolding from a local template directory
- **WHEN** the user runs `relai new <template-dir> <new-dir>` with a local template directory
- **THEN** the system copies the template's files into `<new-dir>` and writes any repo-contract
  file (`AGENTS.md`, `relai.yaml`, `.relai/notes.md`) the template did not already provide

#### Scenario: Scaffolding from a clone URL
- **WHEN** the user runs `relai new <template-url> <new-dir>` with a clone URL
- **THEN** the system fetches the template the same way `relai index build` reads a clone-URL
  entry (shallow, read-only, cleaned up after) and scaffolds `<new-dir>` from it

#### Scenario: Target directory already exists and is not empty
- **WHEN** `<new-dir>` already exists and contains files
- **THEN** the system refuses to scaffold into it and reports the conflict, rather than
  overwriting or merging silently

### Requirement: Template placeholder substitution
The system SHALL substitute a documented, fixed set of placeholder tokens (at minimum the new
repo's name) in the template's text files while copying them.

#### Scenario: Repo name substituted into a template file
- **WHEN** a template text file contains the documented repo-name placeholder token
- **THEN** the scaffolded copy in `<new-dir>` has that token replaced with the new repo's name,
  derived from `<new-dir>`'s base name unless the user overrides it

#### Scenario: Binary or non-text template files are copied unchanged
- **WHEN** a template file is not text (for example an image)
- **THEN** the system copies it byte for byte, without attempting placeholder substitution

### Requirement: Scaffolding never overwrites existing repo-contract content
The system SHALL follow the same never-overwrite rule as `relai init`: a repo-contract file the
template already provides is left as the template wrote it, and `relai init`'s own templates fill
in only what is still missing.

#### Scenario: Template already provides AGENTS.md
- **WHEN** the template includes its own `AGENTS.md`
- **THEN** the system does not replace it with `relai init`'s default content

### Requirement: Relai ships no built-in template
The system SHALL require an explicit template source on every `relai new` invocation and SHALL
NOT bundle a default template of its own.

#### Scenario: Running relai new without a template argument
- **WHEN** the user runs `relai new` with no template source given
- **THEN** the system reports that a template source is required and does not scaffold anything

### Requirement: Scaffolding points to the cross-repo index
After a successful scaffold, the system SHALL print the command to add the new repo to the
group's cross-repo index, without running it automatically.

#### Scenario: Successful scaffold suggests indexing
- **WHEN** `relai new` finishes scaffolding `<new-dir>` successfully
- **THEN** the system's output names `relai index build` as the next step to make the new repo
  discoverable via `relai find`
