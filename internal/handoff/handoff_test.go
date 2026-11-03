package handoff

import (
	"fmt"
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
	_ = os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Agents\n"), 0o644)
	g("add", "AGENTS.md")
	g("commit", "-q", "-m", "add AGENTS.md")
	for i := 0; i < commits; i++ {
		_ = os.WriteFile(filepath.Join(dir, "f.txt"), []byte(strings.Repeat("x", i+1)), 0o644)
		g("add", "f.txt")
		g("commit", "-q", "-m", "commit number "+strings.Repeat("long message ", 5))
	}
	return dir
}

// bareGitRepo is a git repo with none of AGENTS.md/CLAUDE.md, openspec/, or .beads/ — degraded
// mode should kick in. The single commit is backdated well past the 7-day recent-files window,
// so tests can add their own recent commits deliberately instead of inheriting "just now".
func bareGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	g := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
			"GIT_AUTHOR_DATE=2020-01-01T00:00:00", "GIT_COMMITTER_DATE=2020-01-01T00:00:00")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	g("init", "-q", "-b", "main")
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0o644)
	g("add", "README.md")
	g("commit", "-q", "-m", "init")
	return dir
}

func TestDegradedMode_GitSectionUnchanged(t *testing.T) {
	base := sources{
		branch:  "main",
		head:    "abc123 2026-01-01",
		commits: []string{"abc123 2026-01-01 init", "def456 2026-01-02 feat: x"},
		dirty:   []string{" M foo.go"},
	}
	full, degraded := base, base
	full.degraded, degraded.degraded = false, true

	fullDoc := render("repo", full, 8, 8, 0, 0, 0, -1)
	degradedDoc := render("repo", degraded, 8, 8, 0, 0, 0, -1)

	stateSection := func(doc string) string {
		i := strings.Index(doc, "## State")
		j := strings.Index(doc[i:], "\n## ")
		return doc[i : i+j]
	}
	if got, want := stateSection(degradedDoc), stateSection(fullDoc); got != want {
		t.Errorf("State section differs:\nfull:\n%s\ndegraded:\n%s", want, got)
	}
	if !strings.Contains(degradedDoc, "degraded mode") {
		t.Errorf("expected degraded mode label:\n%s", degradedDoc)
	}
}

func TestDegradedMode_NoTasksOrActiveChanges(t *testing.T) {
	doc, err := Build(bareGitRepo(t), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"## Tasks", "## Active changes"} {
		if strings.Contains(doc, unwanted) {
			t.Errorf("degraded mode must omit %q entirely:\n%s", unwanted, doc)
		}
	}
}

func TestDegradedMode_RecentFilesAndMarkers(t *testing.T) {
	dir := bareGitRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("// TODO: fix this\npackage main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", "main.go")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "-m", "add main.go")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Recently touched files (last 7 days)", "`main.go`", "## Open markers (TODO/FIXME)", "TODO: fix this"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
}

func TestDegradedMode_EmptyListsStateNoneFound(t *testing.T) {
	doc, err := Build(bareGitRepo(t), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Recently touched files (last 7 days)\n(none)", "## Open markers (TODO/FIXME)\n(none found)"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
}

func TestDegradedMode_LongListsAreTruncated(t *testing.T) {
	dir := bareGitRepo(t)
	g := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, fmt.Sprintf("// TODO: item %d padded with extra words to eat the budget faster", i))
	}
	_ = os.WriteFile(filepath.Join(dir, "many.go"), []byte(strings.Join(lines, "\n")+"\npackage main\n"), 0o644)
	g("add", "many.go")
	g("commit", "-q", "-m", "add many TODOs")

	o := DefaultOptions()
	o.MaxTokens = 400
	doc, err := Build(dir, o)
	if err != nil {
		t.Fatal(err)
	}
	if got := EstimateTokens(doc); got > o.MaxTokens {
		t.Fatalf("handoff is %d tokens, budget %d", got, o.MaxTokens)
	}
	if !strings.Contains(doc, "more)") {
		t.Errorf("expected a truncation marker:\n%s", doc)
	}
}

func TestBuildBasics(t *testing.T) {
	dir := gitRepo(t, 3)
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Handoff — ", "Branch `main`", "Recent commits:", "(format v0.2)", "no task graph", "## Harness", "no `.relai/notes.md` yet"} {
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
