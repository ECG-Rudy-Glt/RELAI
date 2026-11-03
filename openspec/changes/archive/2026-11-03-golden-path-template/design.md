# Design

## Context

See `proposal.md` — Why. Relevant existing state:

- `internal/index` already has the exact sourcing logic this change needs: read a local path, or
  shallow-clone a clone URL to a temp dir read-only, clean up after (`internal/index/index.go`,
  `cloneShallow`/`readOne`). This change reuses that logic rather than duplicating it.
- `internal/initrepo` already writes the repo contract (`AGENTS.md`, `relai.yaml`,
  `.relai/notes.md`) without ever overwriting an existing file (`Init`, `exists` checks). This
  change calls it for anything the template doesn't already provide, rather than reimplementing
  "never overwrite".
- The project is Go, standard library only unless a dependency is clearly worth it (`AGENTS.md`).
  Token substitution needs nothing beyond `text/template` and `io/fs`.

## Goals / Non-Goals

**Goals:**
- Make starting a new repo from a group's own pre-wired template a single command, without Relai
  prescribing what that template contains.
- Reuse existing sourcing (`internal/index`) and contract-writing (`internal/initrepo`) logic
  instead of growing a second implementation of either.
- Keep it git-only, one binary, no server — consistent with "easy first".

**Non-Goals:**
- Shipping any actual template (lab, framework, language-specific or otherwise). `docs/DESIGN.md`
  §9 is explicit that lab-specific details stay out of this repo; the same reasoning extends to
  any opinionated template, which would privilege one stack and contradict "vendor-neutral".
- Automatically registering the new repo in the cross-repo index. `relai index build` already
  requires an explicit list file (`adoption-and-cross-repo-index`); scaffolding should not create
  a side effect on a file it doesn't own. The command only prints the next step.
- A template *authoring* tool (validating placeholder syntax, linting a template repo). Templates
  are plain directories with a documented token convention; authoring one is on the group, not on
  Relai.
- Proving the DESIGN.md §12 Phase 4 acceptance criterion ("new tool in 15 minutes, in the index,
  deployed") — that needs a real template and real CI/hosting, both outside this repo. It is the
  owner's validation exercise once this ships, not a task of this change.

## Decisions

**Template format: a plain directory tree with `{{.Field}}` tokens in text files, read with Go's
`text/template`.**
Alternative considered: a custom lightweight placeholder syntax (e.g. `__REPO_NAME__`) to avoid
template-engine edge cases (actions, conditionals) appearing by accident in ordinary text files.
Rejected: `text/template`'s delimiters (`{{ }}`) are rare enough in normal source/config files that
accidental collisions are unlikely, it's already in the standard library (no new dependency), and
giving template authors the full power of `text/template` (conditionals, ranges) for free costs
nothing extra to support. A file that fails to parse as a template is copied byte-for-byte
unchanged instead of erroring, so a stray `{{` in an unrelated file degrades gracefully.

**Fixed token set, not a user-supplied key/value map.**
`relai new` substitutes a small, documented set of fields (repo name; owner, from `relai.yaml`
conventions) rather than accepting arbitrary `--var key=value` flags. Alternative considered:
arbitrary variables, like `cookiecutter`/`copier`. Rejected for v1: arbitrary variables need a
prompt/schema mechanism (a `template.yaml` manifest, interactive prompts) that is real scope the
proposal doesn't ask for. A fixed token set covers the proposal's "new tool in 15 minutes" case
(the thing every template needs is its own name) without building a templating sub-system.
Revisit only if a real template hits the limit.

**Template sourcing reuses `internal/index`'s local-path/clone-URL logic directly (same package),
not a copy.**
`internal/index.readOne`-equivalent behavior (stat local path, else shallow-clone a URL to a temp
dir, clean up) is extracted so both `internal/index` and the new scaffolding code call it, instead
of `cmd/relai` growing a second clone-and-cleanup implementation that could drift from the first.

**Target directory must not already exist and contain files; `relai new` never merges into one.**
Alternative considered: merge into an existing directory, like `relai init` does for the contract
files it already writes one at a time. Rejected: a template can contain many files at once, so
partial-merge semantics (which existing file wins, is it overwritten or skipped per-file) are
large additional surface for the same "never overwrite" guarantee that `relai init` gives file by
file. Starting from a non-empty directory is both rare (this is a *new* repo) and better served by
the user scaffolding into a fresh temp directory and merging by hand if they really need to.

**Contract completion after templating reuses `internal/initrepo.Init` unchanged.**
Running `Init` on `<new-dir>` after the template is copied gives the same "write only what's
missing" behavior `relai init` already has and already tests, instead of a parallel "did the
template provide X" check.

## Risks / Trade-offs

- **[Risk]** A template's non-repo-contract text files (source code, CI config) may contain `{{`
  sequences that are not intended as Relai placeholders (e.g. a Go template file, a Jinja config).
  **Mitigation**: only the documented token names resolve; an unrecognized `{{...}}` in a file that
  still parses as a valid `text/template` either resolves to an empty value (danger) or errors.
  Resolved here: parse with `Option("missingkey=error")` so an unrecognized field fails that file's
  substitution, and on any template-parse or -execution error for a given file, copy that file
  byte-for-byte unchanged instead of failing the whole scaffold — see Open Questions for the one
  case this doesn't fully cover.
- **[Risk]** A clone-URL template requires network access and may be slow or fail (private repo,
  offline). **Mitigation**: same as `internal/index`'s existing handling — report the failure
  plainly; nothing is scaffolded on failure (no partial `<new-dir>`).
- **[Trade-off]** No template authoring/validation tooling means a broken template (missing a
  `relai.yaml` convention, inconsistent tokens) fails at scaffold time, per-repo, rather than being
  caught once when the template itself changes. Acceptable: templates are few and owned by the
  group that writes them; add linting later only if this proves painful in practice.

## Open Questions

- Whether a template file that legitimately wants literal `{{` text (rare, but possible in a
  template *about* templating) needs an escape hatch. Doesn't change the spec or the chosen
  approach — default to "no escape hatch in v1," revisit only if a real template hits it.
