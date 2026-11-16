# Design

- **Propose, never declare.** A name in a doc is not a dependency (`portfolio` can be an ordinary word). The tool reports
  evidence and its strength; the agent and the engineer decide. This is why `--propose` prints and `--prompt` asks for confirmation.
- **Strong versus weak.** A remote URL or `../name` is hard to produce by accident; a bare name is not. Short names are
  only searched through strong evidence, to avoid matching everywhere.
- **Weak evidence is filtered, strong evidence never is.** Measured on 14 repositories (eval/results/2026-11-16-discover-links-precision.md):
  a repository's own task, notes, spec and handoff files (`.relai/`, `.beads/`, `openspec/`, TODO, tasks, handoff, passation) are
  not weak evidence, and a name that is one lowercase word of at most 12 letters only counts in a path-like context (`/name`, `name/`,
  `name.git`). `--min-weak` defaults to 2. `--propose` prints the relai.yaml id when the target has one, else the folder name, marked.
- **Generated files are ignored** (`graphify-out`, `node_modules`, `vendor`, lock files, `go.sum`, minified files).
- **Symbolic links are followed** to find repositories (people link projects from elsewhere into one folder) and a repository
  reached twice is listed once.
- **Links versus capabilities.** `provides`/`consumes` say what; `links` says who. Both are checked by a person.
- **Feeds the team server.** RELAI-TEAM (`RELAI-TEAM-9os`) is where a team sees the linked repositories together; the discovery
  is the input it needs. Not part of this change.
