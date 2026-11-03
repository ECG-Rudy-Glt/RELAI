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
	for _, want := range []string{"# Handoff — ", "Branch `main`", "Recent commits:", "(format v0.4)", "no task graph", "## Harness", "no `.relai/notes.md` yet"} {
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

func TestTeamHint(t *testing.T) {
	cases := []struct {
		name                       string
		owners, authors, assignees int
		want                       bool
	}{
		{"solo", 1, 1, 0, false},
		{"one signal: owners", 3, 1, 0, false},
		{"one signal: authors", 1, 4, 1, false},
		{"one signal: assignees", 1, 1, 2, false},
		{"owners and authors", 2, 2, 0, true},
		{"authors and assignees", 1, 2, 2, true},
		{"all three", 2, 3, 2, true},
	}
	for _, c := range cases {
		got := teamHint(c.owners, c.authors, c.assignees)
		if (got != "") != c.want {
			t.Errorf("%s: hint = %q, want present=%v", c.name, got, c.want)
		}
		if c.want && !strings.Contains(got, TeamURL) {
			t.Errorf("%s: hint does not point to relai-team: %q", c.name, got)
		}
	}
	if h := teamHint(2, 3, 0); !strings.Contains(h, "2 owners") || !strings.Contains(h, "3 commit authors") || strings.Contains(h, "assignees") {
		t.Errorf("hint should name only the signals that fired: %q", h)
	}
}

// teamRepo is a git repo with two owners in relai.yaml and commits by two different authors.
func teamRepo(t *testing.T, owners string) string {
	t.Helper()
	dir := gitRepo(t, 0)
	_ = os.WriteFile(filepath.Join(dir, "relai.yaml"), []byte("id: x\nowners: "+owners+"\n"), 0o644)
	g := func(author string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME="+author, "GIT_AUTHOR_EMAIL=a@t", "GIT_COMMITTER_NAME="+author, "GIT_COMMITTER_EMAIL=a@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	g("bob", "add", "relai.yaml")
	g("bob", "commit", "-q", "-m", "add manifest")
	_ = os.WriteFile(filepath.Join(dir, "g.txt"), []byte("y"), 0o644)
	g("carol", "add", "g.txt")
	g("carol", "commit", "-q", "-m", "more")
	return dir
}

func TestHandoffShowsTeamHintOnlyWithTwoSignals(t *testing.T) {
	two, err := Build(teamRepo(t, "[team-a, team-b]"), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(two, "Team signals: 2 owners, 3 commit authors") || !strings.Contains(two, TeamURL) {
		t.Errorf("expected the hint with two signals:\n%s", two)
	}
	one, _ := Build(teamRepo(t, "[team-a]"), DefaultOptions()) // authors only: t, bob, carol
	if strings.Contains(one, "Team signals") {
		t.Errorf("one signal must not show the hint:\n%s", one)
	}
	solo, _ := Build(gitRepo(t, 2), DefaultOptions())
	if strings.Contains(solo, "Team signals") {
		t.Errorf("a solo repo must not show the hint:\n%s", solo)
	}
}

func TestTeamHintIsFirstCutUnderBudget(t *testing.T) {
	dir := teamRepo(t, "[team-a, team-b]")
	full, _ := Build(dir, DefaultOptions())
	if !strings.Contains(full, "Team signals") {
		t.Fatal("setup: hint missing")
	}
	// A budget just under the full document must drop the hint and nothing else of substance.
	o := DefaultOptions()
	o.MaxTokens = EstimateTokens(full) - 1
	cut, _ := Build(dir, o)
	if strings.Contains(cut, "Team signals") {
		t.Errorf("hint should be the first cut:\n%s", cut)
	}
	if !strings.Contains(cut, "Recent commits:") || !strings.Contains(cut, "add manifest") {
		t.Errorf("cutting the hint should not cost commits:\n%s", cut)
	}
}

// fakeBD puts a `bd` script first on PATH. Using it keeps these tests independent of the real tool.
func fakeBD(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "bd"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

const snapshotJSONL = `{"_type":"issue","id":"p-1","title":"Fix the login redirect","status":"open","priority":1,"issue_type":"task"}
{"_type":"issue","id":"p-2","title":"Write the migration","status":"in_progress","priority":0,"issue_type":"task"}
{"_type":"issue","id":"p-3","title":"Old thing","status":"closed","priority":2,"issue_type":"task"}
{"_type":"issue","id":"p-4","title":"The epic","status":"open","priority":0,"issue_type":"epic"}
{"_type":"issue","id":"p-5","title":"Deferred idea","status":"deferred","priority":3,"issue_type":"task"}
`

func repoWithBeadsFiles(t *testing.T, jsonl string) string {
	dir := gitRepo(t, 1)
	_ = os.MkdirAll(filepath.Join(dir, ".beads"), 0o755)
	if jsonl != "" {
		_ = os.WriteFile(filepath.Join(dir, ".beads", "issues.jsonl"), []byte(jsonl), 0o644)
	}
	return dir
}

func TestFreshCloneWithoutDatabaseSaysSoInsteadOfZeroTasks(t *testing.T) {
	fakeBD(t, `echo "Error: no beads database found" >&2; exit 1`)
	doc, err := Build(repoWithBeadsFiles(t, snapshotJSONL), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Tasks cannot be read here: this clone has no Beads database yet",
		"`bd bootstrap --yes`",
		"snapshot kept in git",
		"`p-2` P0 Write the migration", // in progress, listed first
		"Open (1 total):",
		"`p-1` P1 Fix the login redirect",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
	for _, bad := range []string{"Ready (unblocked, 0 total)", "Old thing", "The epic", "Deferred idea"} {
		if strings.Contains(doc, bad) {
			t.Errorf("must not contain %q (a closed/epic/deferred task, or the misleading zero):\n%s", bad, doc)
		}
	}
}

func TestFreshCloneWithoutSnapshotStillSaysHowToLoadTasks(t *testing.T) {
	fakeBD(t, `echo "Error: no beads database found" >&2; exit 1`)
	doc, _ := Build(repoWithBeadsFiles(t, ""), DefaultOptions())
	if !strings.Contains(doc, "Tasks cannot be read here") || !strings.Contains(doc, "bd bootstrap --yes") {
		t.Errorf("no explanation without a snapshot:\n%s", doc)
	}
	if strings.Contains(doc, "snapshot kept in git") || strings.Contains(doc, "Ready (unblocked") {
		t.Errorf("claims things it does not have:\n%s", doc)
	}
}

func TestOtherBdFailuresAreReportedWithoutRawOutput(t *testing.T) {
	fakeBD(t, `echo "panic: secret internal detail /home/x/token=abc" >&2; exit 2`)
	doc, _ := Build(repoWithBeadsFiles(t, snapshotJSONL), DefaultOptions())
	if !strings.Contains(doc, "bd failed to read the tasks") {
		t.Errorf("failure not reported:\n%s", doc)
	}
	if strings.Contains(doc, "secret internal detail") || strings.Contains(doc, "token=abc") {
		t.Errorf("raw tool output leaked into the handoff:\n%s", doc)
	}
}

func TestBdNotInstalledFallsBackToTheSnapshot(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git")
	}
	only := t.TempDir()
	if err := os.Symlink(git, filepath.Join(only, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", only) // git, and nothing else: no bd, no openspec
	doc, _ := Build(repoWithBeadsFiles(t, snapshotJSONL), DefaultOptions())
	if !strings.Contains(doc, "the bd command is not installed") || !strings.Contains(doc, "`p-1` P1 Fix the login redirect") {
		t.Errorf("not installed case:\n%s", doc)
	}
}

func TestHealthyBdIsUnchangedByTheFallback(t *testing.T) {
	fakeBD(t, `case "$1" in ready) echo '[{"id":"live-1","title":"Live task","priority":1}]';; *) echo '[]';; esac`)
	doc, _ := Build(repoWithBeadsFiles(t, snapshotJSONL), DefaultOptions())
	if !strings.Contains(doc, "`live-1` P1 Live task") || !strings.Contains(doc, "Ready (unblocked, 1 total)") {
		t.Errorf("live tasks not shown:\n%s", doc)
	}
	if strings.Contains(doc, "cannot be read") || strings.Contains(doc, "Fix the login redirect") {
		t.Errorf("the snapshot must not appear when bd works:\n%s", doc)
	}
}
