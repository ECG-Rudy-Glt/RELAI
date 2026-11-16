# Proposal

## Why

Relai's design (`docs/DESIGN.md` §2, §11) and `docs/multi-repo.md` §9 ("not a service") commit to
"git only, one binary, no server, no portal" for a person or small team. That constraint is what
makes Relai adoptable on day one, and it stays unchanged. But a team working across several repos
of one ecosystem still wants a live, shared view of work and claim/release coordination between
agents — something the git-only CLI deliberately does not provide. Rather than adding that to this
repo (which would turn "git-only, no server" into "git-only, except..."), it is built as a
**separate, optional project**, [`relai-team`](https://github.com/ECG-Rudy-Glt/RELAI-TEAM), that
depends on Relai's repo contract (`relai.yaml`, Beads, OpenSpec) without Relai ever depending on it.
This change only updates Relai's own docs to state that positioning clearly, so the "not a service"
claim in `docs/multi-repo.md` and the verdict in `docs/DESIGN.md` §11 don't read as contradicted by
`relai-team`'s existence.

## What Changes

- Update `docs/DESIGN.md` §11 (market-scan verdict) to note that a team-coordination server is
  being explored as a separate, optional project (`relai-team`) rather than inside Relai, and why
  (see `design.md`).
- Add a short pointer in `docs/DESIGN.md` §12 (plan) linking to `relai-team` for anyone tracking
  Relai's roadmap, without adding a phase whose deliverables live in another repo's own plan.
- Scope `docs/multi-repo.md` §9 ("not a service") explicitly to this repo, with a one-line pointer
  to `relai-team` for the server use case, so the statement stays true rather than silently
  outdated.
- No code, CLI, `relai.yaml` schema, or behavior change in this repo. `skip_specs: true` is set on
  this change accordingly.

## Capabilities

### New Capabilities
(none — this change is documentation only; the `team-server`, `mcp-team-interface`, and
`deployment-presets` capabilities are specified and implemented in the separate `relai-team`
repository, not here)

### Modified Capabilities
(none)

## Impact

- **Code**: none.
- **Docs**: `docs/DESIGN.md` §11 and §12, `docs/multi-repo.md` §9.
- **Non-code**: none.
- **Tracking**: Beads epic `relai-idg` in this repo, scoped to this doc update only; the
  implementation work is tracked in `relai-team`'s own `.beads/`.
