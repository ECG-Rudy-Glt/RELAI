# Tasks

## 1. Cross-repo index: build

- [x] 1.1 Add `internal/index` with a function that reads a list of repo paths/URLs and parses
      each one's `relai.yaml`, skipping and reporting entries that are missing, not a git repo, or
      have no `relai.yaml`; verify with a unit test covering one valid local repo, one missing
      path, and one repo with no `relai.yaml`.
- [x] 1.2 Support clone-URL entries by shallow-cloning to a temp dir before reading, cleaning up
      after; verify with a test using a local `file://` URL to a fixture repo (no network needed
      in tests).
- [x] 1.3 Add `relai index build <list-file> [-o <output>]` to `cmd/relai`, writing the collected
      `index.json` (default path documented in `README.md`); verify by running it against a small
      fixture list and diffing the resulting `index.json`.
- [x] 1.4 Document the index file format and `repos.txt`-style input in `README.md`; verify the
      documented command and file format match what 1.1-1.3 implement.

## 2. Cross-repo index: find

- [x] 2.1 Add `relai find <capability> [--index <path>]` to `cmd/relai`: read `index.json`, print
      every entry whose `relai.yaml` `provides:` lists that capability (id, path/URL, summary), or
      state plainly that none was found; verify with a unit test against a fixture `index.json`
      covering both outcomes.
- [x] 2.2 Handle a missing index file with a message naming `relai index build`, not a raw
      file-not-found error; verify with a test running `relai find` with no `index.json` present.
- [x] 2.3 Document `relai find` usage in `README.md` with the two outcomes from 2.1; verify the
      documented example output matches actual command output.

## 3. Degraded-mode handoff (git-only)

- [x] 3.1 Reuse `relai discover`'s harness-presence inventory (already in `internal/initrepo`) to
      detect "no `AGENTS.md`/`CLAUDE.md`, no `openspec/`, no `.beads/`"; verify with a unit test on
      a fixture repo with none of the three present.
- [x] 3.2 Add a degraded-mode branch to `internal/handoff` that keeps the existing git-state
      section (branch, latest commit, uncommitted count, recent commits) unchanged and labels the
      report as degraded mode; verify with a test asserting the git section's content matches
      full-mode output for the same repo state.
- [x] 3.3 Add the recently-touched-files section (files changed in the last 7 days via `git log
      --since`) and the `TODO`/`FIXME` section (via `git grep`) to degraded mode, reusing the
      existing budget-truncation helper and reporting how many entries were omitted when
      truncated; verify with tests for: both lists present, both empty (reports "none found"
      rather than omitting the section), and a long-list truncation case.
- [x] 3.4 Verify degraded mode never emits a Tasks or Active changes section when no tracker/specs
      are present, with a test asserting their absence (not an empty placeholder).
- [x] 3.5 Document degraded mode in `docs/HANDOFF.md` and `spec/handoff.md`; verify the documented
      example matches actual output on a fixture bare-git repo.

## 4. Release and public install

- [x] 4.1 Add a goreleaser config building Linux/macOS/Windows binaries for `cmd/relai`; verify
      with a local `goreleaser release --snapshot --clean` run producing the expected archives.
- [x] 4.2 Add a GitHub Actions release workflow triggered on version tags, running the existing
      `gofmt`/`vet`/`test` checks before invoking goreleaser; verify by inspecting a dry run (or a
      tag pushed to a fork) completes and attaches binaries to the release.
- [x] 4.3 Add a one-line install script (shell) that downloads the right binary for the host OS/
      arch from the latest release; verify by running it in a clean container/VM and checking
      `relai version` resolves afterward.
- [x] 4.4 Update `README.md`'s Quick start with the install script and binary download as
      alternatives to `go install`; verify the documented commands work as written.

## 5. Public proof content

- [ ] 5.1 Draft a post adapting `docs/DESIGN.md` §11 (market scan and verdict) and
      `eval/README.md` (the 1.6-16x token figures and the 96,000 -> 1,700 real-repo case) for an
      external audience, linking the eval harness so the numbers are independently reproducible;
      verify by having the owner review the draft before publishing.
- [ ] 5.2 Publish the post and share it where the target audience is (e.g. Hacker News, an agent-
      tooling community); verify by confirming the post is live and linked from `README.md`.

## 6. Hosted-index design note (documentation only)

- [x] 6.1 Add a short section to `docs/DESIGN.md` documenting the optional, deferred hosted/Docker
      index service described in `design.md` - explicitly additive, off by default, not built in
      this change; verify the section states it is deferred and links back to this change.

## Workflow follow-up

- Archive this change once tasks 1-4 and 6 are complete and reviewed by the owner; task 5 (public
  content) may land after archive since it changes no code or spec.
- Update Beads: close or update `relai-a7o.8` (release) and `relai-d6i` (multi-repo) to reflect
  what this change actually delivered versus what remains (hub, sync, managed blocks still open).
