// Package initrepo adds the Relai repo contract to an existing repository without
// overwriting anything that is already there.
package initrepo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Result lists what Init did, for display.
type Result struct {
	Created []string
	Skipped []string
}

// Init writes AGENTS.md (or links it to an existing CLAUDE.md), relai.yaml and .relai/notes.md.
func Init(dir string, linkClaude bool) (Result, error) {
	var r Result
	dir, err := filepath.Abs(dir)
	if err != nil {
		return r, err
	}
	name := slug(filepath.Base(dir))

	agents := filepath.Join(dir, "AGENTS.md")
	claude := filepath.Join(dir, "CLAUDE.md")
	switch {
	case exists(agents):
		r.Skipped = append(r.Skipped, "AGENTS.md (already present)")
	case exists(claude) && linkClaude:
		if err := os.Symlink("CLAUDE.md", agents); err != nil {
			return r, fmt.Errorf("link AGENTS.md to CLAUDE.md: %w", err)
		}
		r.Created = append(r.Created, "AGENTS.md -> CLAUDE.md")
	default:
		if err := write(agents, agentsTemplate(name, exists(claude))); err != nil {
			return r, err
		}
		r.Created = append(r.Created, "AGENTS.md")
	}

	for _, f := range []struct{ path, body string }{
		{filepath.Join(dir, "relai.yaml"), manifestTemplate(name)},
		{filepath.Join(dir, ".relai", "notes.md"), notesTemplate},
	} {
		rel, _ := filepath.Rel(dir, f.path)
		if exists(f.path) {
			r.Skipped = append(r.Skipped, rel+" (already present)")
			continue
		}
		if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
			return r, err
		}
		if err := write(f.path, f.body); err != nil {
			return r, err
		}
		r.Created = append(r.Created, rel)
	}
	return r, nil
}

func agentsTemplate(name string, hasClaude bool) string {
	extra := ""
	if hasClaude {
		extra = "\nProject-specific instructions also live in `CLAUDE.md`.\n"
	}
	return fmt.Sprintf(`# %s — instructions for AI agents

Start here:
1. Run `+"`relai handoff`"+` (current state, next tasks, decisions, pitfalls — a few hundred tokens).
2. Load anything else only when you need it.
%s
## Rules
- Keep this file under 100 lines.
- Record decisions and pitfalls in `+"`.relai/notes.md`"+`, not in chat.
`, name, extra)
}

func manifestTemplate(name string) string {
	return fmt.Sprintf(`apiVersion: relai.dev/v0
id: %s
summary: ""            # one sentence an agent can use to decide whether to reuse this repo
owners: []
provides: []           # - capability: pdf.generate.invoice
consumes: []           # - capability: auth.oidc
`, name)
}

const notesTemplate = `### Decisions
<!-- One line each: date — decision — why. -->

### Pitfalls
<!-- Things that cost time once and will again: symptom → cause → what to do. -->
`

func write(path, body string) error { return os.WriteFile(path, []byte(body), 0o644) }

func exists(p string) bool { _, err := os.Lstat(p); return err == nil }

var nonSlug = regexp.MustCompile(`[^a-z0-9-]+`)

func slug(s string) string {
	return strings.Trim(nonSlug.ReplaceAllString(strings.ToLower(s), "-"), "-")
}
