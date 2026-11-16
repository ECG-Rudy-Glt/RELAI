package hub

import (
	"fmt"
	"os"
	"path/filepath"
)

const ciHeader = "# Installed once by `relai sync` from the hub. Edit it freely: relai never overwrites it.\n"

// ciGitHub and ciForgejo are the same job: Forgejo and Gitea Actions read the GitHub syntax.
const ciGitHub = ciHeader + `name: relai
on:
  push:
    branches: [main]
  pull_request:
jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install relai
        run: |
          curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh -o install.sh
          RELAI_INSTALL_DIR="$HOME/.local/bin" sh install.sh
          echo "$HOME/.local/bin" >> "$GITHUB_PATH"
      - name: Contract and drift from the hub
        run: relai check
      - name: A newer hub version exists (informational)
        run: relai check --outdated || true
`

const ciGitLab = `# Not tested. Add to .gitlab-ci.yml:
relai:
  image: alpine:3
  before_script:
    - apk add --no-cache git curl
    - curl -fsSL https://raw.githubusercontent.com/ECG-Rudy-Glt/RELAI/main/scripts/install.sh -o install.sh
    - RELAI_INSTALL_DIR="$HOME/.local/bin" sh install.sh
    - export PATH="$HOME/.local/bin:$PATH"
  script:
    - relai check
`

const conventionsTemplate = `# Conventions

Plain rules every repository of the group follows. Keep this file under 100 lines: it is read by
agents at every session.

- Commit messages are conventional (feat:, fix:, docs:).
- Never commit secrets; read them from the environment.
- Write decisions and pitfalls in .relai/notes.md, not in chat.
`

const skillTemplate = `---
name: commit-style
description: How commits are written in this group. Use before committing.
---

# Commit style

Use conventional commits: feat:, fix:, docs:, test:, chore:. One logical change per commit, the
subject in the imperative, under 72 characters.
`

const readmeTemplate = `# Hub

Shared conventions of the group, vendored by ` + "`relai sync`" + ` into each repository.

- ` + "`CONVENTIONS.md`" + `: the rules (under 100 lines).
- ` + "`skills/<name>/`" + `: shared agent skills, installed as ` + "`hub-<name>`" + `.
- ` + "`ci/`" + `: CI job templates, installed once.
- ` + "`repos.txt`" + `: one clone URL per line, for ` + "`relai index build`" + `.

Release a version by tagging it (v1.0.0). A repository pins it with
` + "`conventions: <this repository>@v1.0.0`" + ` in relai.yaml, then runs ` + "`relai sync --apply`" + `.
`

// InitHub scaffolds a hub in dir, which must be absent or empty. It returns the files created.
func InitHub(dir string) ([]string, error) {
	if entries, err := os.ReadDir(dir); err == nil {
		if len(entries) > 0 {
			return nil, fmt.Errorf("%s is not empty: a hub is created in an empty directory", dir)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	files := []struct{ path, data string }{
		{"CONVENTIONS.md", conventionsTemplate},
		{"README.md", readmeTemplate},
		{"skills/commit-style/SKILL.md", skillTemplate},
		{"ci/github.yml", ciGitHub},
		{"ci/forgejo.yml", ciGitHub},
		{"ci/gitlab.yml", ciGitLab},
		{"repos.txt", "# one clone URL or path per line\n"},
	}
	var created []string
	for _, f := range files {
		p := filepath.Join(dir, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return created, err
		}
		if err := os.WriteFile(p, []byte(f.data), 0o644); err != nil {
			return created, err
		}
		created = append(created, f.path)
	}
	return created, nil
}
