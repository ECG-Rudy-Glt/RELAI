package handoff

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGH puts a gh that runs script first on PATH.
func fakeGH(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func githubRepo(t *testing.T) string {
	t.Helper()
	dir := gitRepo(t, 1)
	cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/example/demo.git")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote add: %v\n%s", err, out)
	}
	return dir
}

// askForIssues writes the relai.yaml line that turns the GitHub adapter on.
func askForIssues(t *testing.T, dir string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "relai.yaml"), []byte("id: demo\nsummary: A demo.\ntasks: github\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const ghIssuesJSON = `[{"number":12,"title":"Login redirect loops","assignees":[{"login":"alice"}],"labels":[]},` +
	`{"number":13,"title":"Export to CSV","assignees":[],"labels":[{"name":"feature"},{"name":"p1"}]},` +
	`{"number":14,"title":"Typo in the README","assignees":[],"labels":[]}]`

func TestGitHubIssuesFillTheTasksSection(t *testing.T) {
	dir := askForIssues(t, githubRepo(t))
	fakeGH(t, "cat <<'JSON'\n"+ghIssuesJSON+"\nJSON")
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"From GitHub Issues", "Assigned:", "- `#12` Login redirect loops — @alice",
		"Open, unassigned (2 total):", "- `#13` Export to CSV [feature, p1]", "- `#14` Typo in the README",
		"Details: `gh issue view <number>`.",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "no task graph") {
		t.Errorf("issues were read, the placeholder must not be printed:\n%s", doc)
	}
}

func TestGitHubFailureIsSaidNotHidden(t *testing.T) {
	dir := askForIssues(t, githubRepo(t))
	fakeGH(t, "echo 'secret raw error' >&2; exit 1")
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "GitHub Issues were not read: gh could not list the issues") {
		t.Errorf("a failing gh must be reported with a fixed reason:\n%s", doc)
	}
	if strings.Contains(doc, "secret raw error") {
		t.Errorf("raw tool output leaked into the handoff:\n%s", doc)
	}
}

// The handoff makes no network call unless the repository asks for it.
func TestGitHubIsNotAskedUnlessTheRepositorySaysSo(t *testing.T) {
	fakeGH(t, "echo called > \"$GH_MARK\"; echo '[]'")
	mark := filepath.Join(t.TempDir(), "mark")
	t.Setenv("GH_MARK", mark)
	doc, err := Build(githubRepo(t), DefaultOptions()) // a GitHub remote, but no `tasks: github`
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "(no task graph in this repo; to list the open GitHub issues here, add `tasks: github` to `relai.yaml`)") {
		t.Errorf("a GitHub repository must be told how to turn the adapter on:\n%s", doc)
	}
	t.Setenv("RELAI_NO_GH", "1")
	doc, err = Build(askForIssues(t, githubRepo(t)), DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(mark); err == nil {
		t.Error("gh was run without `tasks: github`, or with RELAI_NO_GH set")
	}
	if !strings.Contains(doc, "(no task graph in this repo)") {
		t.Errorf("expected the plain placeholder:\n%s", doc)
	}
}

func TestJSONHandoffCarriesTheSameFacts(t *testing.T) {
	dir := askForIssues(t, githubRepo(t))
	fakeGH(t, "cat <<'JSON'\n"+ghIssuesJSON+"\nJSON")
	if err := os.MkdirAll(filepath.Join(dir, ".relai"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, ".relai", "notes.md"), []byte("### Decisions\n- 2026-01-01 — a decision\n"), 0o644)
	raw, err := BuildJSON(dir)
	if err != nil {
		t.Fatal(err)
	}
	var d JSONDoc
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, raw)
	}
	if d.Format != FormatVersion || d.Repo != filepath.Base(dir) || d.State.Branch != "main" || len(d.State.Commits) != 2 {
		t.Errorf("state is wrong: %+v", d)
	}
	if d.Tasks.Source != "github" || len(d.Tasks.InProgress) != 1 || d.Tasks.InProgress[0].ID != "#12" || len(d.Tasks.Open) != 2 {
		t.Errorf("tasks are wrong: %+v", d.Tasks)
	}
	if len(d.Notes) != 2 || d.Notes[1] != "- 2026-01-01 — a decision" {
		t.Errorf("notes are wrong: %v", d.Notes)
	}
	// Lists a consumer iterates over are [] when empty, never null.
	for _, key := range []string{`"ready": []`, `"blocked": []`, `"changes": []`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("expected %s in:\n%s", key, raw)
		}
	}
}
