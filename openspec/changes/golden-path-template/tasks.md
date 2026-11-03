# Tasks

## 1. Shared template-sourcing logic

- [x] 1.1 Extract `internal/index`'s local-path/clone-URL sourcing (stat a local path; else
      shallow-clone a URL to a temp dir, read-only, cleaned up after) into a function both
      `internal/index` and the new scaffolding code call, with no behavior change; verify
      `internal/index`'s existing tests still pass unchanged.

## 2. Scaffolding core

- [x] 2.1 Add `internal/scaffold` with a function that copies a template directory to a new target
      directory, refusing if the target already exists and is non-empty; verify with a unit test
      covering a fresh target and an existing non-empty target.
- [x] 2.2 Substitute the documented token set (at minimum repo name, derived from the target
      directory's base name) in each copied text file using `text/template` with
      `Option("missingkey=error")`; on a parse or execution error for a file, copy that file
      byte-for-byte unchanged instead of failing the whole scaffold; verify with tests for: a text
      file with a valid token, a binary file (copied unchanged), and a file with an unresolvable
      `{{...}}` (copied unchanged, scaffold still succeeds).
- [x] 2.3 After copying, call `internal/initrepo.Init` on the target directory so any repo-contract
      file (`AGENTS.md`, `relai.yaml`, `.relai/notes.md`) the template didn't provide is written,
      without overwriting what the template did provide; verify with a test asserting a
      template-provided `AGENTS.md` is left untouched while a missing `relai.yaml` is written.

## 3. CLI

- [x] 3.1 Add `relai new <template> <dir>` to `cmd/relai`, wiring local-path and clone-URL template
      sources through the task 1.1 sourcing function and task 2.* scaffolding; require an explicit
      template argument (no default template); verify by running it against a small fixture
      template directory and diffing the scaffolded output.
- [x] 3.2 On success, print the exact `relai index build` command as the next step, without running
      it; verify with a test asserting the command's output names `relai index build`.
- [x] 3.3 On a missing template argument or an unreadable/missing template source, print a clear
      error naming the problem instead of a raw stack trace or file-not-found error; verify with a
      test for each case (no argument; template path does not exist).

## 4. Documentation

- [x] 4.1 Document the template token convention and a worked example (a minimal fixture template,
      the exact `relai new` command, and the resulting repo) in `README.md`; verify the documented
      command and output match what tasks 2-3 implement.
- [x] 4.2 Mark adoption level 3 ("Golden path") as shipped in `docs/DESIGN.md` §5, and add a short
      note to §12 Phase 4 pointing at this change; verify the table and phase note read correctly
      against the shipped command.

## Workflow follow-up

- File a new Beads epic for this change once the proposal is reviewed (no existing task covers
  it — see proposal.md Impact).
- Archive this change once tasks 1-4 are complete and reviewed by the owner.
- The DESIGN.md §12 Phase 4 acceptance criterion ("new tool in 15 minutes, in the index,
  deployed") is a validation exercise for the owner with a real template and real CI/hosting
  outside this repo — not a tracked task here.
