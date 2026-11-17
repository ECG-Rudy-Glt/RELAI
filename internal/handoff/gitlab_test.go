package handoff

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeGlab puts a glab that runs script first on PATH.
func fakeGlab(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "glab"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func gitlabRepo(t *testing.T, host string) string {
	t.Helper()
	dir := gitRepo(t, 1)
	cmd := exec.Command("git", "remote", "add", "origin", "https://"+host+"/example/demo.git")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, out)
	}
	return dir
}

func askForGitLab(t *testing.T, dir string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "relai.yaml"), []byte("id: demo\nsummary: A demo.\ntasks: gitlab\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// The shape of the GitLab REST issue object, which `glab issue list --output json` prints (extra
// fields are there on purpose: they must be ignored).
const glIssuesJSON = `[{"id":901,"iid":12,"state":"opened","title":"Login redirect loops","labels":[],"web_url":"https://gitlab.com/example/demo/-/issues/12",` +
	`"assignees":[{"id":1,"username":"alice","name":"Alice"}]},` +
	`{"id":902,"iid":13,"state":"opened","title":"Export to CSV","labels":["feature","p1"],"assignees":[]},` +
	`{"iid":14,"title":"Typo in the README"},` +
	`{"iid":15,"state":"closed","title":"Already done"}]`

func TestGitLabIssuesFillTheTasksSection(t *testing.T) {
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	fakeGlab(t, "cat <<'JSON'\n"+glIssuesJSON+"\nJSON")
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"From GitLab Issues (open issues, read with `glab`)", "Assigned:", "- `#12` Login redirect loops — @alice",
		"Open, unassigned (2 total):", "- `#13` Export to CSV [feature, p1]", "- `#14` Typo in the README",
		"Details: `glab issue view <number>`.",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "Already done") || strings.Contains(doc, "no task graph") {
		t.Errorf("closed issue listed, or placeholder printed:\n%s", doc)
	}
}

// One call, with the documented flags, and nothing else.
func TestGitLabIsCalledOnceWithTheDocumentedFlags(t *testing.T) {
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	log := filepath.Join(t.TempDir(), "log")
	t.Setenv("GL_LOG", log)
	fakeGlab(t, "echo \"$@\" >> \"$GL_LOG\"; echo '[]'")
	if _, err := Build(dir, DefaultOptions()); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(log)
	if got := strings.TrimSpace(string(b)); got != "issue list --output json --per-page 50" {
		t.Errorf("glab was run as %q", got)
	}
}

func TestGitLabFailuresAreSaidNotHidden(t *testing.T) {
	cases := []struct{ name, script, want string }{
		{"not signed in", "echo 'secret raw error' >&2; exit 1", "GitLab Issues were not read: glab could not list the issues"},
		{"unreadable output", "echo 'secret raw output'", "GitLab Issues were not read: glab returned output that could not be read"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
			fakeGlab(t, c.script)
			doc, err := Build(dir, DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(doc, c.want) {
				t.Errorf("missing %q in:\n%s", c.want, doc)
			}
			if strings.Contains(doc, "secret raw") || strings.Contains(doc, "Ready: 0") {
				t.Errorf("raw tool output or an empty list leaked into the handoff:\n%s", doc)
			}
		})
	}
}

func TestGitLabAbsentIsSaid(t *testing.T) {
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	empty := t.TempDir()
	// Keep git on PATH (a symlink), hide glab.
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git")
	}
	if err := os.Symlink(gitPath, filepath.Join(empty, "git")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", empty)
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "GitLab Issues were not read: the glab command is not installed") {
		t.Errorf("a missing glab must be reported:\n%s", doc)
	}
}

func TestGitLabTimeoutIsBounded(t *testing.T) {
	old := issueTimeout
	issueTimeout = 200 * time.Millisecond
	t.Cleanup(func() { issueTimeout = old })
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	fakeGlab(t, "exec sleep 5")
	start := time.Now()
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 4*time.Second {
		t.Error("the handoff waited on a hanging glab")
	}
	if !strings.Contains(doc, "GitLab Issues were not read: glab could not list the issues") {
		t.Errorf("a timeout must be reported with a fixed reason:\n%s", doc)
	}
}

func TestGitLabIsNotAskedUnlessTheRepositorySaysSo(t *testing.T) {
	mark := filepath.Join(t.TempDir(), "mark")
	t.Setenv("GL_MARK", mark)
	fakeGlab(t, "echo called > \"$GL_MARK\"; echo '[]'")
	doc, err := Build(gitlabRepo(t, "gitlab.example.org"), DefaultOptions()) // GitLab remote, no `tasks: gitlab`
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "(no task graph in this repo; to list the open GitLab issues here, add `tasks: gitlab` to `relai.yaml`)") {
		t.Errorf("a GitLab repository must be told how to turn the adapter on:\n%s", doc)
	}
	t.Setenv("RELAI_NO_GH", "1")
	doc, err = Build(askForGitLab(t, gitlabRepo(t, "gitlab.com")), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Error("glab was run without `tasks: gitlab`, or with RELAI_NO_GH set")
	}
	if !strings.Contains(doc, "(no task graph in this repo)") {
		t.Errorf("expected the plain placeholder:\n%s", doc)
	}
}

// Beads wins: a .beads/ directory means glab is not asked, like gh.
func TestGitLabIsIgnoredWithBeads(t *testing.T) {
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	if err := os.MkdirAll(filepath.Join(dir, ".beads"), 0o755); err != nil {
		t.Fatal(err)
	}
	mark := filepath.Join(t.TempDir(), "mark")
	t.Setenv("GL_MARK", mark)
	fakeGlab(t, "echo called > \"$GL_MARK\"; echo '[]'")
	if _, err := Build(dir, DefaultOptions()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Error("glab was run in a repository with .beads/")
	}
}

func TestGitLabTitlesAreDataAndControlCharactersAreStripped(t *testing.T) {
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	fakeGlab(t, `printf '[{"iid":7,"title":"Fix \\u001b[31mred\\u0007 now","labels":[],"assignees":[]}]'`)
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(doc, "\x1b\x07") {
		t.Errorf("control characters in the handoff: %q", doc)
	}
	if !strings.Contains(doc, "`#7` Fix [31mred now") {
		t.Errorf("title text lost:\n%s", doc)
	}
	raw, err := BuildJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(string(raw), "\x1b\x07") || strings.Contains(string(raw), `\u001b`) {
		t.Errorf("control characters in the JSON handoff: %s", raw)
	}
}

func TestGitLabBudgetKeepsTheListShort(t *testing.T) {
	var items []string
	for i := 1; i <= 50; i++ {
		items = append(items, `{"iid":`+strings.Repeat("1", 1)+string(rune('0'+i%10))+`,"title":"Task number `+strings.Repeat("x", 20)+`","labels":[],"assignees":[]}`)
	}
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	fakeGlab(t, "cat <<'JSON'\n["+strings.Join(items, ",")+"]\nJSON")
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "Open, unassigned (50+ total):") {
		t.Errorf("a full page must say 50+:\n%s", doc)
	}
	if n := strings.Count(doc, "Task number"); n >= 50 {
		t.Errorf("the list is not capped by the budget: %d lines", n)
	}
}

func TestGitLabJSONNamesTheBackend(t *testing.T) {
	dir := askForGitLab(t, gitlabRepo(t, "gitlab.com"))
	fakeGlab(t, "cat <<'JSON'\n"+glIssuesJSON+"\nJSON")
	raw, err := BuildJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	var d JSONDoc
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatal(err)
	}
	if d.Format != "0.10" || d.Tasks.Source != "gitlab" || len(d.Tasks.InProgress) != 1 || d.Tasks.InProgress[0].ID != "#12" || len(d.Tasks.Open) != 2 {
		t.Errorf("tasks are wrong: %+v", d)
	}
}

func TestGitHubRepositoryKeepsItsHint(t *testing.T) {
	doc, err := Build(githubRepo(t), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "add `tasks: github` to `relai.yaml`") || strings.Contains(doc, "GitLab") {
		t.Errorf("GitHub hint changed:\n%s", doc)
	}
}
