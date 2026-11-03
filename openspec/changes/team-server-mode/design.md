# Design

## Context

See `proposal.md` - Why. This document only covers the one decision this change makes: where the
team-coordination capability lives, and why that place is not this repo. The capability's own
architecture (storage, claim semantics, HTTP/MCP surface, deployment presets) is specified and
designed in `relai-team`'s own `openspec/changes/team-server-foundation/design.md`, not here.

## Goals / Non-Goals

**Goals:**
- State, in Relai's own docs, that team coordination is being built as a separate project, so
  `docs/DESIGN.md` §11 and `docs/multi-repo.md` §9 stay accurate.
- Keep this repo's CLI, schema, and behavior completely unchanged.

**Non-Goals:**
- Designing the team server itself — out of scope for this repo; see `relai-team`.
- Deciding `relai-team`'s internals (storage format, HTTP API, MCP library, auth) — that project
  owns those decisions.

## Decisions

**A separate repository, not a mode flag or second binary in this repo.** Earlier drafting of this
change (superseded) considered a `cmd/relai-server` binary inside this repo. Splitting it into its
own repository instead means: (1) this repo's "git-only, no server" claim stays literally true, not
just true-by-default; (2) `relai-team` can depend on Relai's data contract (files) without any Go
module coupling, consistent with this repo's own "compose, don't rebuild" principle; (3) the two
projects can ship, version, and fail or succeed independently, so a rough edge in `relai-team`
never reflects on Relai's own credibility. Alternative considered: a second binary in this repo's
`cmd/` — rejected because it still couples the two projects' release cadence and dependency graph
even if the binaries are separate artifacts.

**Relai's docs link to `relai-team`; Relai does not vendor or re-describe its design.** Keeping
`relai-team`'s architecture decisions in its own repo (not duplicated here) avoids the two drifting
out of sync; this repo's docs only need to state that the capability exists elsewhere and why.

## Risks / Trade-offs

- **A reader of `docs/DESIGN.md` §11 could still read "not a service" as contradicted by a
  same-org project named `relai-team`.** Mitigation: the doc update this change makes states the
  split explicitly, with the rationale, rather than leaving it implicit.
- **Two repos to keep conceptually aligned** (e.g., if Relai's `relai.yaml` schema changes in a way
  `relai-team`'s aggregation depends on). Mitigation: `relai-team` reads Relai's contract via files
  and CLIs (`bd list --json`, `openspec list --json`, `relai.yaml` directly), not via a pinned
  library dependency, so there is no version-lockstep requirement — only the file formats need to
  stay compatible, and those are already Relai's versioned, documented contract.
