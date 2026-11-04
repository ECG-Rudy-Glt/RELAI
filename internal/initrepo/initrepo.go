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

// Options controls what Init sets up.
type Options struct {
	// LinkClaude links a new AGENTS.md to an existing CLAUDE.md instead of writing a fresh one.
	LinkClaude bool
	// ClaudeCode also wires the repo for Claude Code: a CLAUDE.md that imports AGENTS.md and a
	// session-start hook that loads the handoff, so the agent starts from the project's state.
	ClaudeCode bool
}

// Init writes AGENTS.md (or links it to an existing CLAUDE.md), relai.yaml and .relai/notes.md.
func Init(dir string, linkClaude bool) (Result, error) {
	return InitWith(dir, Options{LinkClaude: linkClaude})
}

// InitWith is Init with options. It never overwrites a file that is already there.
func InitWith(dir string, o Options) (Result, error) {
	linkClaude := o.LinkClaude
	var r Result
	dir, err := filepath.Abs(dir)
	if err != nil {
		return r, err
	}
	name := slug(filepath.Base(dir))

	// Check what could block us before writing anything, so a refused run leaves no half-done
	// repo behind (say, AGENTS.md written and then ".relai is a file").
	for _, d := range []string{".relai", ".claude", filepath.Join(".claude", "hooks")} {
		if d != ".relai" && !o.ClaudeCode {
			continue
		}
		if st, err := os.Lstat(filepath.Join(dir, d)); err == nil && !st.IsDir() {
			return r, fmt.Errorf("%s exists and is not a directory: move it away, nothing was written", d)
		}
	}

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
	if o.ClaudeCode {
		if err := wireClaudeCode(dir, &r); err != nil {
			return r, err
		}
	}
	return r, nil
}

// SessionStartScript is the hook Claude Code runs when a session starts: what it prints becomes part
// of the session's context. It never fails, so a broken hook cannot block a session.
const SessionStartScript = `#!/bin/sh
# Runs when a Claude Code session starts (see .claude/settings.json). Whatever it prints becomes part of
# the session's context, so the agent starts from the project's current state instead of guessing.
# It never fails: a broken hook must not block a session.
cd "${CLAUDE_PROJECT_DIR:-.}" 2>/dev/null || exit 0
if command -v relai >/dev/null 2>&1 && relai handoff 2>/dev/null; then
  exit 0
fi
echo "# Handoff (the relai CLI is not installed, so this is read from the plain files)"
for f in .relai/notes.md relai.yaml; do
  if [ -f "$f" ]; then echo; echo "## $f"; cat "$f"; fi
done
exit 0
`

// ClaudeSettings registers SessionStartScript as a SessionStart hook.
const ClaudeSettings = `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          { "type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/session-start.sh", "timeout": 30 }
        ]
      }
    ]
  }
}
`

// wireClaudeCode adds what Claude Code needs to start from the project's state. Claude Code reads
// CLAUDE.md (newer versions also fall back to AGENTS.md, older ones do not), so a CLAUDE.md that
// imports AGENTS.md is the version-proof way; the hook makes loading the handoff automatic instead of
// something the agent has to remember to do.
func wireClaudeCode(dir string, r *Result) error {
	claude := filepath.Join(dir, "CLAUDE.md")
	if exists(claude) {
		r.Skipped = append(r.Skipped, "CLAUDE.md (already present: it should import AGENTS.md with a line `@AGENTS.md`, or be the file AGENTS.md links to)")
	} else {
		if err := write(claude, "@AGENTS.md\n"); err != nil {
			return err
		}
		r.Created = append(r.Created, "CLAUDE.md")
	}
	hook := filepath.Join(dir, ".claude", "hooks", "session-start.sh")
	settings := filepath.Join(dir, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(hook), 0o755); err != nil {
		return err
	}
	if exists(hook) {
		r.Skipped = append(r.Skipped, ".claude/hooks/session-start.sh (already present)")
	} else {
		if err := os.WriteFile(hook, []byte(SessionStartScript), 0o755); err != nil {
			return err
		}
		r.Created = append(r.Created, ".claude/hooks/session-start.sh")
	}
	if exists(settings) {
		r.Skipped = append(r.Skipped, ".claude/settings.json (already present: add a SessionStart hook that runs .claude/hooks/session-start.sh yourself)")
	} else {
		if err := write(settings, ClaudeSettings); err != nil {
			return err
		}
		r.Created = append(r.Created, ".claude/settings.json")
	}
	return nil
}

func agentsTemplate(name string, hasClaude bool) string {
	extra := ""
	if hasClaude {
		extra = "\nProject-specific instructions also live in `CLAUDE.md`.\n"
	}
	return fmt.Sprintf(`# %s — instructions for AI agents

Start here:
1. Run `+"`relai handoff`"+` (current state, next tasks, decisions, pitfalls — a few hundred tokens).
   `+"`relai`"+` not installed or not found? Read `+"`.relai/notes.md`"+`, `+"`relai.yaml`"+`, and
   (if present) `+"`.relai/index.json`"+` directly instead — they are plain files, no CLI required.
2. Before building something another project might already provide, check
   `+"`relai find <capability>`"+` (or read `+"`.relai/index.json`"+` directly) — don't duplicate
   what already exists elsewhere.
3. Load anything else only when you need it.
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
