## ADDED Requirements

### Requirement: Repositories of a folder are found without a list
The system SHALL find the git repositories of a folder, following symbolic links to repositories and listing each repository once, and SHALL say which of them have no `relai.yaml`.

#### Scenario: A folder of projects
- **WHEN** the user runs `relai index build --scan <folder>`
- **THEN** every git repository directly in the folder is indexed when it has a `relai.yaml`, and the others are named as needing `relai init`

#### Scenario: A project linked from elsewhere
- **WHEN** a folder entry is a symbolic link to a git repository
- **THEN** it is found, and a second link to the same repository does not list it twice

### Requirement: Mentions between repositories are reported as evidence, not as links
The system SHALL report, for each pair of repositories, how many files of one mention the other, separating strong evidence (the other's remote URL, a relative path to it) from weak evidence (its name as a word), SHALL show file paths but never file content, SHALL ignore generated files, and SHALL NOT write to any repository or declare a link by itself.

#### Scenario: A deployment mentions the application
- **WHEN** one repository holds a manifest that points at another repository's remote URL
- **THEN** the report shows a strong mention with that file as evidence

#### Scenario: A common word
- **WHEN** a repository's name is a short common word
- **THEN** it is only searched through strong evidence

#### Scenario: Proposal for a person to check
- **WHEN** the user asks for a proposal
- **THEN** a `links:` block per repository is printed with the strength of each line, and nothing is written

### Requirement: Declared links are part of the manifest
The system SHALL read an optional `links:` list of repository ids from `relai.yaml`, keep it in the index, and show it in the workspace overview.

#### Scenario: Declared links shown
- **WHEN** a project declares `links: [portfolio]`
- **THEN** the workspace overview shows that it works with portfolio
