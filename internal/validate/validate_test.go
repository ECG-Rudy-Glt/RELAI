package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

const good = "apiVersion: relai.dev/v0\nid: demo\nsummary: A demo.\nprovides:\n  - capability: demo.thing\n"

func has(f []Finding, level, part string) bool {
	for _, x := range f {
		if x.Level == level && strings.Contains(x.Message, part) {
			return true
		}
	}
	return false
}

func TestAValidRepositoryHasNoFindings(t *testing.T) {
	f := Run(repo(t, map[string]string{"relai.yaml": good, "AGENTS.md": "# demo\n", ".relai/notes.md": "### Decisions\n"}))
	if len(f) != 0 {
		t.Fatalf("expected no findings, got %v", f)
	}
	if Failed(f, true) || !strings.HasPrefix(Text(f), "ok:") {
		t.Errorf("a clean repository must pass, even with --strict: %q", Text(f))
	}
}

func TestMissingFiles(t *testing.T) {
	f := Run(repo(t, nil))
	if !has(f, Error, "not found") || !Failed(f, false) {
		t.Errorf("a missing relai.yaml is an error: %v", f)
	}
	if n := len(f); n != 3 {
		t.Errorf("expected one finding per missing file, got %d: %v", n, f)
	}
}

func TestManifestProblems(t *testing.T) {
	cases := []struct{ name, yaml, level, want string }{
		{"no id", "apiVersion: relai.dev/v0\nsummary: x\n", Error, "id is missing"},
		{"bad id", "apiVersion: relai.dev/v0\nid: my repo\nsummary: x\n", Error, "is not a name"},
		{"empty summary", "apiVersion: relai.dev/v0\nid: a\nsummary: \"\"\n", Error, "summary is missing"},
		{"long summary", "apiVersion: relai.dev/v0\nid: a\nsummary: " + strings.Repeat("x", 201) + "\n", Warning, "one sentence"},
		{"no version", "id: a\nsummary: x\n", Warning, "apiVersion is missing"},
		{"future version", "apiVersion: relai.dev/v1\nid: a\nsummary: x\n", Error, "not a version this relai reads"},
		{"typo in a key", good + "provide: [a]\n", Warning, `unknown key "provide"`},
		{"value not understood", "apiVersion: relai.dev/v0\nid: a\nsummary: x\nprovides: nonsense\n", Error, "was not understood"},
		{"entry with no capability", "apiVersion: relai.dev/v0\nid: a\nsummary: x\nprovides:\n  - name: x\n", Error, "no capability"},
		{"capability name", "apiVersion: relai.dev/v0\nid: a\nsummary: x\nprovides: [Auth OIDC]\n", Warning, "dotted lowercase"},
		{"capability twice", "apiVersion: relai.dev/v0\nid: a\nsummary: x\nprovides: [a.b, a.b]\n", Warning, "listed twice"},
		{"key twice", good + "id: other\n", Error, `"id" appears twice`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := Run(repo(t, map[string]string{"relai.yaml": c.yaml, "AGENTS.md": "x\n", ".relai/notes.md": "x\n"}))
			if !has(f, c.level, c.want) {
				t.Errorf("want a %s containing %q, got %v", c.level, c.want, f)
			}
			if c.level == Warning && Failed(f, false) {
				t.Errorf("a warning alone must not fail without --strict: %v", f)
			}
			if !Failed(f, true) {
				t.Errorf("--strict must fail on any finding: %v", f)
			}
		})
	}
}

func TestAgentsBudget(t *testing.T) {
	files := map[string]string{"relai.yaml": good, ".relai/notes.md": "x\n"}
	files["AGENTS.md"] = strings.Repeat("line\n", MaxAgentsLines)
	if f := Run(repo(t, files)); len(f) != 0 {
		t.Errorf("%d lines is within the budget: %v", MaxAgentsLines, f)
	}
	files["AGENTS.md"] = strings.Repeat("line\n", MaxAgentsLines) + "one more"
	if f := Run(repo(t, files)); !has(f, Error, "101 lines") {
		t.Errorf("101 lines is over the budget: %v", f)
	}
}

func TestInfoNeverFails(t *testing.T) {
	f := []Finding{{Level: Info, File: "AGENTS.md", Message: "declared override"}}
	if Failed(f, false) || Failed(f, true) {
		t.Error("an info must never fail, even with --strict")
	}
	f = append(f, Finding{Level: Warning, File: "x", Message: "w"})
	if Failed(f, false) || !Failed(f, true) {
		t.Error("a warning fails only with --strict")
	}
	if got := Text(f); !strings.Contains(got, "0 error(s), 1 warning(s), 1 info.") {
		t.Errorf("Text: %q", got)
	}
}
