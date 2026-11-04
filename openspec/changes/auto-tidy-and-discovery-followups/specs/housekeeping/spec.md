## ADDED Requirements

### Requirement: Merged work closes its tasks and tidies by itself
The system SHALL, on request or after a pull or merge, close the tasks that a commit on the default branch says it closes with an explicit keyword and the task's id, refresh the committed task export without the owner field, and delete local branches that are fully merged, SHALL leave open any task that is only named, SHALL NOT close anything from a branch other than the default one, SHALL NOT commit, and SHALL NOT fail a commit or a pull because of it.

#### Scenario: A merged commit says it closes a task
- **WHEN** the default branch holds a commit whose message says "Closes p-1" and p-1 is open
- **THEN** p-1 is closed with a reason naming the commit, and `p-10` is not touched

#### Scenario: A task is only named
- **WHEN** a commit names a task without a closing keyword
- **THEN** the task stays open and tidy reports it

#### Scenario: On a feature branch
- **WHEN** tidy runs on a branch that is not the default one
- **THEN** no task is closed

#### Scenario: Unmerged branch
- **WHEN** a local branch has commits that are not in the default branch
- **THEN** it is kept

### Requirement: Hooks are bounded
The system SHALL give `bd` a time limit in the pre-commit hook it installs and SHALL let a commit or pull go through when that limit, a missing tool or a failure is hit.

#### Scenario: bd hangs
- **WHEN** `bd` does not answer within the limit during a commit
- **THEN** the commit completes
