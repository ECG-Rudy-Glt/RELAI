# Proposal

## Why

Relai's design and E2 eval (docs/DESIGN.md, eval/README.md) show the repo-contract handoff works:
as correct as reading the full docs, 1.6-16x fewer tokens, and a measured 96,000 -> 1,700 token
drop on a real infra repo. But three gaps stand between that result and real adoption:

1. There is no installable release and no public proof point — the only way to try Relai today is
   `go install` from source, and the before/after token numbers live only in `eval/README.md`.
2. The handoff alone is "too easy to copy" (DESIGN.md §11, risks) — any agent vendor can ship an
   equivalent natively. The one gap the market scan found nobody else covers is cross-repo
   anti-duplication (`docs/multi-repo.md`, task `relai-d6i`), and it does not exist yet.
3. `relai handoff` today assumes the composed ecosystem (`AGENTS.md`, OpenSpec, Beads) is present.
   A team with none of that gets no value, which narrows adoption to teams already using that
   exact stack.

This change closes gaps 2 and 3 with real spec'd behavior, and packages gap 1 (release + public
proof) as execution work, so Relai has something installable, something hard to copy, and
something useful to a bare-git team, before investing in outreach.

## What Changes

- Add a cross-repo index: a flat collection of `relai.yaml` files (one per project) that
  `relai find <capability>` searches before a new capability is built, surfacing existing
  `provides:` entries that already cover it (relai-d6i groundwork, docs/multi-repo.md §6 already
  names this command; this change specs and implements it).
- Add a degraded handoff mode: when `AGENTS.md`, OpenSpec, and Beads are all absent,
  `relai handoff` still produces a useful summary from bare git alone (recent commits, branches,
  recently touched files, `TODO`/`FIXME` grep) instead of an empty or near-empty report.
- Ship an installable release: goreleaser binaries for Linux/macOS/Windows, a one-line install
  script, referenced from the README (relai-a7o.8, already tracked in Beads).
- Publish the market scan (DESIGN.md §11) and the E2 eval results (eval/README.md) as external
  content (a post, not shipped code) making the before/after token numbers legible outside this
  repo.
- Explore, as a documented option only, a hosted/Docker variant of the cross-repo index for teams
  that want a shared index without each person cloning the index repo locally. This stays
  additive and never the default — see design.md; it does not change the "not a portal" decision
  in DESIGN.md and is not implemented in this change.

## Capabilities

### New Capabilities
- `cross-repo-index`: collecting `relai.yaml` files from known projects into a queryable index and
  checking a capability name against it before new work starts.
- `git-only-handoff`: producing a bounded, useful `relai handoff` report from a bare git
  repository when none of `AGENTS.md`, OpenSpec, or Beads is present.

### Modified Capabilities
(none — no existing `openspec/specs/` capabilities exist yet in this repo)

## Impact

- **Code**: `cmd/relai` (new `find` subcommand, index reader), `internal/handoff` (degraded-mode
  branch when harness detection in `internal/initrepo`-style discovery finds nothing), a new
  `internal/index` package. No changes to `relai.yaml` schema beyond what `provides:`/`consumes:`
  already defines.
- **Docs**: `docs/DESIGN.md` (record the hosted-index option as an open/deferred question),
  `docs/multi-repo.md` (supersede the proposal status once `relai check` exists), `README.md`
  (install instructions, `relai check` usage).
- **Non-code**: a blog/HN post using existing `DESIGN.md` §11 and `eval/README.md` numbers; a
  goreleaser config and CI release workflow. These are tracked as tasks, not specs, since they do
  not change system behavior.
- **Tracking**: extends existing Beads tasks `relai-d6i` (multi-repo) and `relai-a7o.8` (release);
  no new epic needed.
