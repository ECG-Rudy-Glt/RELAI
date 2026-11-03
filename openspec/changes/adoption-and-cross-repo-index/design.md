# Design

## Context

See `proposal.md` - Why. Relevant existing state:

- `internal/handoff` already builds the full-mode report deterministically from git, Beads, and
  OpenSpec, with budget enforcement by truncation (docs/HANDOFF.md, decisions section). This
  change adds a degraded branch to the same package, it does not replace the existing one.
- `docs/multi-repo.md` already designed the hub/index/sync system (phase 2) at a scope larger than
  this change: a hub repo, managed blocks, `relai sync`, multi-CI templates, `relai check`
  (contract/drift validation, distinct from the index). This change implements only the index
  slice of that design (`relai index build`, `relai find`), pull-based and read-only, without the
  hub, managed blocks, or sync machinery. `relai check` (contract validation) stays out of scope;
  it is tracked separately as `relai-a7o.5`.
- `relai.yaml` already has `provides:`/`consumes:` with a `capability:` field (see this repo's own
  `relai.yaml`). No schema change is needed for the index to work; `spec/relai.yaml.schema.json`
  (`relai-a7o.1`) should eventually formalize it but is not a dependency of this change.

## Goals / Non-Goals

**Goals:**
- Make the one gap the market scan found uncovered (cross-repo anti-duplication) real and
  demoable with a minimal index: build + find, nothing else.
- Make `relai handoff` useful on day one in a repo with zero prior Relai/AGENTS.md/OpenSpec/Beads
  adoption, so the tool's value does not require the rest of the ecosystem first.
- Keep both additions git-only, one-binary, no server - consistent with the project's "easy
  first" constraint (AGENTS.md, DESIGN.md decisions).

**Non-Goals:**
- The hub repo, managed blocks, `relai sync`, and multi-CI templates from `docs/multi-repo.md`
  stay out of scope; they depend on decisions (open questions §10 of that doc) not yet made by the
  owner.
- A hosted/Docker index service is explored below as an option, not built in this change.
- `relai check` (contract/drift validation, `relai-a7o.5`) is a separate, already-tracked task;
  this change does not touch it, despite the proposal's earlier wording risking confusion with it.
- Degraded-mode handoff does not try to infer tasks or decisions heuristically (e.g. parsing
  commit messages for a tracker-like structure) - that risks guessing wrong silently, which
  conflicts with the project's "deterministic handoff" decision. It only reports what git and the
  working tree state directly.

## Decisions

**Index storage format: one JSON file, not a database.**
A single `index.json` (array of `{source: <path-or-url>, relai_yaml: <parsed content>}`) next to
the list of repos. Alternative considered: SQLite - rejected, adds a dependency and a binary
format to a project that is "stdlib only" and wants every file to stay greppable/diffable in git.

**Index building is a separate step from finding (`relai index build` then `relai find`), not a
live search.**
Alternative considered: `relai find` re-reading every repo on each call. Rejected: that requires
every repo to be freshly cloned/fetched at call time, which does not hold for a clone URL list
(`docs/multi-repo.md`'s pull-based hub builds on a schedule) and would make `relai find` slow and
network-dependent for no benefit when repos are local clones that move at human speed.

**`relai index build` accepts local paths and clone URLs in the same list.**
A URL is fetched to a temporary shallow clone (read-only), a local path is read directly. This
matches both the single-person case (a few repos cloned side by side) and the team case
(`repos.txt` of clone URLs in `docs/multi-repo.md`) without two separate commands.

**Degraded-mode detection is a single "harness present?" check reused from `internal/initrepo`'s
discovery, not a new detector.**
`relai discover` (already shipped, per `docs/HANDOFF.md`) already inventories
`AGENTS.md`/`CLAUDE.md`, OpenSpec, and Beads presence. Degraded mode triggers when that inventory
is empty of all three, so there is one source of truth for "is a harness present" shared by
`discover` and `handoff`.

**Degraded mode is additive output, not a different code path with its own budget logic.**
It reuses `internal/handoff`'s existing truncate-to-budget helper on its own sections (recent
files, TODO/FIXME matches) rather than writing a second truncation implementation.

**Hosted/Docker index service: documented as an open, deferred option - not built now.**
The request to "explore" this is honored by writing it into `docs/DESIGN.md` as an explicitly
optional, additive future layer (a small read-only HTTP server serving the same `index.json` a
team already builds with `relai index build`, for teams that would rather query an endpoint than
clone an index repo). It must stay optional and off by default: the CLI's `relai find` keeps
working against a local `index.json` with no server involved, which is what keeps the "not a
portal" decision (DESIGN.md v0.2) intact. No server code is written in this change - only the
design note. Building it is left as a future task once a team actually asks for it, so the
project does not speculatively build server/Docker infrastructure nobody requested yet.

## Risks / Trade-offs

- **[Risk]** A clone-URL index entry requires network access and may be slow or fail (private
  repo, offline). **Mitigation**: same as "a listed repo is missing" in the spec - skip and report
  the reason, never fail the whole build.
- **[Risk]** Degraded mode's `TODO`/`FIXME` grep could be noisy on large repos (thousands of
  matches). **Mitigation**: same truncation budget as the rest of the handoff; the spec requires
  stating how many were omitted rather than silently cutting.
- **[Risk]** Publishing the market scan and eval numbers externally (proposal's content work)
  invites scrutiny of methodology (sample size: 4 fixtures + a handful of real repos).
  **Mitigation**: this is a content/process risk, not a spec risk - link the eval harness
  (`eval/README.md`) so the numbers are independently reproducible rather than asserted.
- **[Trade-off]** Building the index as a flat JSON file instead of a richer store means no
  incremental/partial rebuild yet (full rebuild each `relai index build`). Acceptable at the
  single-team scale this change targets; revisit only if `docs/multi-repo.md`'s scheduled-hub-job
  scenario shows it matters.

## Open Questions

- Exact token budget split between the two degraded-mode lists (recently-touched files vs.
  TODO/FIXME matches) when both are long - the spec requires truncating "the longer list first"
  but leaves the precise per-section cap to implementation, same granularity as existing
  full-mode truncation rules.
- Whether `relai index build`'s temporary clones for URL entries should be cached between runs for
  speed - does not change the spec or the command's observable behavior, safe to decide during
  implementation.
