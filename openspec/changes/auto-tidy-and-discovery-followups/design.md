# Design

- **Evidence, not inference.** A task is closed only when a commit message on the default branch holds a closing keyword
  (close/closes/closed, fix/fixes/fixed, resolve/resolves/resolved) followed by an id that is open in the export. A task named
  without a keyword is reported and left open: it may be half done. Ids are matched whole (`p-1` is not `p-10`).
- **Default branch only.** On a feature branch the work is not merged, so nothing is closed.
- **No commit.** The export is rewritten and the person (or the next commit, through the pre-commit hook) commits it.
- **Branches use `git branch -d`**, which refuses anything not fully merged; the default branch and `main`/`master`/`develop`
  are never touched; remote branches are never touched.
- **Hooks never fail a commit or a pull** and bound the time they give to other tools.
- **Config evidence.** A repo named in CI, deployment or configuration files (workflows, k8s, compose, yaml, tf) is far more
  often a real dependency than one named in a document, so two such files are enough to propose it. Measured on the owner's
  folder, where documents-only mentions of a hub repo were mostly notes and tests.
