package discover

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScan(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "CLAUDE.md", "x")
	write(t, dir, ".claude/skills/deploy/SKILL.md", "x")
	write(t, dir, ".claude/commands/opsx/apply.md", "x")
	write(t, dir, ".claude/settings.json", `{"hooks":{"SessionStart":[],"PreToolUse":[]},"permissions":{}}`)
	write(t, dir, ".mcp.json", `{"mcpServers":{"homelab":{}}}`)
	write(t, dir, "docs/journal/2026-11.md", "x")
	write(t, dir, "docs/journal/2026-10.md", "x")
	write(t, dir, "docs/HANDOFF.md", "x")
	write(t, dir, "docs/adr/0001-x.md", "x")
	write(t, dir, ".github/workflows/ci.yml", "x")
	write(t, dir, "node_modules/pkg/TODO.md", "x")
	_ = os.MkdirAll(filepath.Join(dir, "openspec"), 0o755)

	inv := Scan(dir)
	want := map[string]string{
		Instructions: "CLAUDE.md",
		Skills:       "deploy",
		Commands:     "/opsx:apply",
		Hooks:        "PreToolUse, SessionStart",
		MCP:          "homelab",
		Specs:        "OpenSpec (openspec/, `openspec list`)",
		Decisions:    "docs/adr/",
		Docs:         "docs/HANDOFF.md, docs/journal/",
		CI:           ".github/workflows/ci.yml",
	}
	for kind, items := range want {
		g := inv.Get(kind)
		if g == nil {
			t.Errorf("%s: not found", kind)
			continue
		}
		if got := strings.Join(g.Items, ", "); got != items {
			t.Errorf("%s: got %q, want %q", kind, got, items)
		}
	}
	if inv.Get(Tasks) != nil {
		t.Errorf("no task tracking expected, got %v", inv.Get(Tasks).Items)
	}
	prompt := Prompt(inv)
	for _, s := range []string{"docs/HANDOFF.md", "### Decisions", "No secrets"} {
		if !strings.Contains(prompt, s) {
			t.Errorf("prompt misses %q", s)
		}
	}
}

func TestHasCoreHarness(t *testing.T) {
	bare := t.TempDir()
	if Scan(bare).HasCoreHarness() {
		t.Error("bare dir: want no core harness")
	}

	withAgents := t.TempDir()
	write(t, withAgents, "AGENTS.md", "x")
	if !Scan(withAgents).HasCoreHarness() {
		t.Error("AGENTS.md present: want core harness")
	}

	withOpenSpec := t.TempDir()
	_ = os.MkdirAll(filepath.Join(withOpenSpec, "openspec"), 0o755)
	if !Scan(withOpenSpec).HasCoreHarness() {
		t.Error("openspec/ present: want core harness")
	}

	withBeads := t.TempDir()
	_ = os.MkdirAll(filepath.Join(withBeads, ".beads"), 0o755)
	if !Scan(withBeads).HasCoreHarness() {
		t.Error(".beads present: want core harness")
	}
}

func TestLinesTruncates(t *testing.T) {
	inv := Inventory{Groups: []Group{{Kind: Skills, Where: ".claude/skills", Items: []string{"a", "b", "c"}}}}
	if got := inv.Lines(2)[0]; got != "Skills (.claude/skills): a, b, … (1 more)" {
		t.Fatalf("got %q", got)
	}
}

// A project's own state, pitfalls and decision files are found, at the root first, while a plan
// deeper down whose name contains "changelog" comes after them and look-alike names do not count.
// Shaped after a real repository where only that plan was listed.
func TestStateAndDecisionFilesAreFoundRootFirst(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{
		"ETAT.md", "DECISIONS.md", "reference/PIEGES.md", "plans/a1b_changelog_texte_2026-10-28.md",
		"docs/PROJECT_STATUS.md", "statement.md", "estate.md", "src/state.go", "reference/PLATEFORME.md",
	} {
		p := filepath.Join(dir, f)
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := map[string][]string{}
	for _, g := range Scan(dir).Groups {
		got[g.Kind] = g.Items
	}
	if d := strings.Join(got[Decisions], ", "); d != "DECISIONS.md" {
		t.Errorf("decision records = %q, want DECISIONS.md", d)
	}
	want := "ETAT.md, docs/PROJECT_STATUS.md, plans/a1b_changelog_texte_2026-10-28.md, reference/PIEGES.md"
	if d := strings.Join(got[Docs], ", "); d != want {
		t.Errorf("docs = %q\nwant  %q (root first; statement.md, estate.md and code files are not state docs)", d, want)
	}
	if l := Scan(dir).Lines(1); !strings.Contains(strings.Join(l, "\n"), "Docs to read on demand: ETAT.md, …") {
		t.Errorf("cut to one item, the root state file must be the one kept:\n%s", strings.Join(l, "\n"))
	}
}
