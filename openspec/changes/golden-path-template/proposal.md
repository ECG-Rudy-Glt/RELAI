# Proposal

## Why

`docs/DESIGN.md` names a "golden path" (adoption level 3, §5) as the fast way to start a new
project already wired with the repo contract, CI and deployment — but it has stayed a line in the
design doc, never built. Today, starting a new tool inside a group that has standardized on its
own CI/hosting means assembling `AGENTS.md`, `relai.yaml`, `.relai/notes.md`, CI config and
deployment manifests by hand, every time, with nothing enforcing that the result is wired the same
way as the group's other repos. `relai init` (level 1) already gives one new repo the contract;
there is no equivalent that starts a repo *from* a group's own pre-wired template.

## What Changes

- Add `relai new <template> <dir>`: scaffolds a new repo at `<dir>` from a template (a local path
  or clone URL, sourced the same way `relai index build` already reads repo lists), substituting a
  small set of placeholder tokens (repo name, owner), then runs the same contract-writing logic as
  `relai init` for anything the template didn't already provide (never overwrites).
- Define the template format as a convention, not a shipped artifact: any directory or repo with
  placeholder tokens in its text files. Relai does not ship an opinionated template — a group's CI
  and deployment choices are its own (`docs/DESIGN.md` §5: "the group's own CI/hosting"), and the
  author's own lab template (Forgejo+Actions, k3s+FluxCD, Traefik, OIDC, Harbor, Prometheus,
  `docs/DESIGN.md` §9) stays out of this repo, per that section's own "lab-specific details stay
  out of this repo" note.
- After scaffolding, print the next step to register the new repo in the group's cross-repo index
  (`relai index build`, shipped in `adoption-and-cross-repo-index`) so it is immediately
  discoverable via `relai find` — ties the golden path into the index instead of duplicating it.
- Document the template convention and a worked example in `README.md`.

## Capabilities

### New Capabilities
- `golden-path-scaffold`: lets a person or team start a new repo pre-wired with the Relai repo
  contract from a reusable template (their own, not one Relai ships), in one command.

### Modified Capabilities
(none)

## Impact

- **Code**: `cmd/relai` (new `new` subcommand), a new `internal/scaffold` package (template
  sourcing reuses `internal/index`'s local-path/clone-URL/shallow-clone logic; token substitution
  uses the standard library's `text/template`, no new dependency), `internal/initrepo` (reused for
  anything the template leaves unwritten).
- **Docs**: `README.md` (template convention, worked example), `docs/DESIGN.md` (mark adoption
  level 3 as shipped once this lands).
- **Out of scope**: any specific template (lab, framework, language) — this change ships the
  scaffolding mechanism only. Acceptance demo ("new tool in 15 minutes, in the index, deployed",
  `docs/DESIGN.md` §12 Phase 4) depends on a real template and real CI/hosting, both outside this
  repo; it is a validation exercise for the owner, not a deliverable of this change.
- **Tracking**: no existing Beads task covers this (it was tracked only as a `docs/DESIGN.md`
  phase) — a new epic should be filed once this proposal is reviewed.
