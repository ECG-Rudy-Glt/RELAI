# Design: decisions taken on 2026-11-16 (not yet implemented)

1. **Pin on one line**: `conventions: <hub>@<tag>` (clone URL or local path, split on the last `@`).
   The format has no nested mappings. All shared skills are installed; no per-repo selection yet.
2. **Conventions are vendored as a whole file**, `.relai/conventions.md`, starting with one generated
   marker line. Not pasted into `AGENTS.md`: its 100-line budget, enforced by `relai validate`, would
   not hold a 100-line `CONVENTIONS.md` plus local rules.
3. **`AGENTS.md` gets a three-line managed block** between `<!-- relai:conventions <tag> … -->` and
   `<!-- /relai:conventions -->` that points to that file; inserted before the first `## `, or at the
   end. Nothing outside the markers is touched. Written through a symlink. Not created if missing.
4. **Shared skills**: hub `skills/<name>/` is copied to `.claude/skills/hub-<name>/` and
   `.agents/skills/hub-<name>/`; a `hub-*` folder no longer in the hub is removed. Symlinks in the hub
   are skipped and reported; nothing is written outside the repository.
5. **CI template once**: `ci/github.yml` to `.github/workflows/relai.yml`, `ci/forgejo.yml` to
   `.forgejo/` or `.gitea/`, only when absent. GitLab: print the lines to add, never edit the file.
6. **Shared hooks are not synced** in this version (merging JSON settings is invasive).
7. **`relai sync` is a dry run** unless `--apply`; it never commits and never opens a pull request.
8. **Fetching**: `git clone --depth 1 --branch <tag>` into the user cache directory, reused when
   present, `--refresh` to fetch again. A missing hub or tag is a clear error, no raw git output.
9. **`relai check`** = the findings of `relai validate` + the handoff fits its budget + (when pinned)
   the vendored file, the block and the `hub-*` skills are identical to the hub at the pinned tag.
   Lines of `AGENTS.md` starting with `Override:` are listed at a new level "info" that never fails.
   `--outdated` warns when a higher version tag exists (`git ls-remote --tags`).
10. **`relai hub init`** refuses a non-empty directory; CI templates are constants in the binary,
    install relai with `scripts/install.sh`, then run `relai check`. The GitLab one is untested.
11. **Layout**: `internal/hub` (pure functions returning a plan), all CLI code in `cmd/relai/hub.go`,
    two small edits in `main.go`.
12. **Left for later**: shared hooks, pull-request automation, per-repo skill selection, the richer
    workspace view (`relai-d6i.5`), the "upstream changes" section (`relai-a7o.4`).

Open point to look at while building: a hostile hub (path traversal in file names, symlinks, very
large files). Treat hub content as untrusted, as `relai new` treats a template.
