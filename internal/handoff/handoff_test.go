package handoff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepo(t *testing.T, commits int) string {
	t.Helper()
	dir := t.TempDir()
	g := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	g("init", "-q", "-b", "main")
	for i := 0; i < commits; i++ {
		_ = os.WriteFile(filepath.Join(dir, "f.txt"), []byte(strings.Repeat("x", i+1)), 0o644)
		g("add", "f.txt")
		g("commit", "-q", "-m", "commit number "+strings.Repeat("long message ", 5))
	}
	return dir
}

func TestBuildBasics(t *testing.T) {
	dir := gitRepo(t, 3)
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Handoff — ", "Branch `main`", "Recent commits:", "no Beads task graph", "no `.relai/notes.md` yet"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
}

func TestBudgetIsEnforced(t *testing.T) {
	dir := gitRepo(t, 15)
	_ = os.MkdirAll(filepath.Join(dir, ".relai"), 0o755)
	notes := strings.Repeat("- a pitfall that is described at some length to eat the budget\n", 200)
	_ = os.WriteFile(filepath.Join(dir, ".relai", "notes.md"), []byte(notes), 0o644)

	o := DefaultOptions()
	o.MaxTokens = 400
	doc, err := Build(dir, o)
	if err != nil {
		t.Fatal(err)
	}
	if got := EstimateTokens(doc); got > o.MaxTokens {
		t.Fatalf("handoff is %d tokens, budget %d", got, o.MaxTokens)
	}
	if !strings.Contains(doc, "more lines in `.relai/notes.md`") {
		t.Errorf("truncated notes should say where the rest is:\n%s", doc)
	}
}

func TestNextTask(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tasks.md")
	_ = os.WriteFile(p, []byte("## 1. Limiter\n- [x] 1.1 Bucket\n\n## 2. HTTP\n  - [ ] 2.1 Middleware: 429\n- [ ] 2.2 Config\n"), 0o644)
	if got := nextTask(p); got != "2.1 Middleware: 429" {
		t.Fatalf("got %q", got)
	}
	if got := nextTask(filepath.Join(t.TempDir(), "missing.md")); got != "" {
		t.Fatalf("missing file: got %q", got)
	}
}

func TestNamedInCommits(t *testing.T) {
	s := sources{
		commits: []string{"9101f28 2026-06-05 feat: run on Postgres 17 (pg-4)", "e6c30be 2026-06-04 feat: pg-40 column"},
		ready:   []task{{ID: "pg-4", Title: "Migrate"}, {ID: "pg-5", Title: "Healthz"}},
		blocked: []task{{ID: "pg-6", Title: "Backfill", BlockedBy: []string{"pg-4"}}},
	}
	got := namedInCommits(s)
	if len(got) != 1 || !strings.HasPrefix(got[0], "`pg-4` Migrate, in 9101f28") {
		t.Fatalf("got %q", got)
	}
}

func TestNotARepo(t *testing.T) {
	if _, err := Build(t.TempDir(), DefaultOptions()); err == nil {
		t.Fatal("expected an error outside a git repository")
	}
}
