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

func TestLinesTruncates(t *testing.T) {
	inv := Inventory{Groups: []Group{{Kind: Skills, Where: ".claude/skills", Items: []string{"a", "b", "c"}}}}
	if got := inv.Lines(2)[0]; got != "Skills (.claude/skills): a, b, … (1 more)" {
		t.Fatalf("got %q", got)
	}
}
