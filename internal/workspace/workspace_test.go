package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

// project makes a git repo with a relai.yaml, optionally a Beads export, committed on a given date.
func project(t *testing.T, root, id, date string, provides []string, jsonl string) index.Entry {
	t.Helper()
	dir := filepath.Join(root, id)
	_ = os.MkdirAll(filepath.Join(dir, ".beads"), 0o755)
	if jsonl != "" {
		_ = os.WriteFile(filepath.Join(dir, ".beads", "issues.jsonl"), []byte(jsonl), 0o644)
	}
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte(id), 0o644)
	env := append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_AUTHOR_DATE="+date+"T12:00:00Z", "GIT_COMMITTER_DATE="+date+"T12:00:00Z")
	for _, a := range [][]string{{"init", "-q", "-b", "main"}, {"add", "-A"}, {"commit", "-q", "-m", "x"}} {
		c := exec.Command("git", a...)
		c.Dir, c.Env = dir, env
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", a, err, out)
		}
	}
	m := index.Manifest{ID: id, Summary: "The " + id + " project."}
	for _, p := range provides {
		m.Provides = append(m.Provides, index.Capability{Capability: p})
	}
	return index.Entry{Source: dir, RelaiYAML: m}
}

const tasks = `{"id":"%[1]s-1","title":"Fix the login redirect","status":"open","priority":1,"issue_type":"task"}
{"id":"%[1]s-2","title":"Second thing","status":"in_progress","priority":2,"issue_type":"task"}
{"id":"%[1]s-3","title":"Closed thing","status":"closed","priority":1,"issue_type":"task"}
{"id":"%[1]s-e","title":"An epic","status":"open","priority":0,"issue_type":"epic"}
`

func TestOverviewIsOrderedByActivityAndCountsFromTheFiles(t *testing.T) {
	root := t.TempDir()
	entries := []index.Entry{
		project(t, root, "old-app", "2025-01-01", nil, ""),
		project(t, root, "billing", "2026-11-03", []string{"pdf.render", "invoice.send"}, fmt.Sprintf(tasks, "b")),
		project(t, root, "auth", "2026-10-20", []string{"auth.oidc"}, ""),
	}
	// A bd on PATH that would fail the test if it were ever run: the overview reads files only.
	fakeBin := t.TempDir()
	_ = os.WriteFile(filepath.Join(fakeBin, "bd"), []byte("#!/bin/sh\ntouch "+filepath.Join(fakeBin, "bd-was-run")+"\nexit 1\n"), 0o755)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	doc := Render(Load(entries), DefaultOptions())
	if _, err := os.Stat(filepath.Join(fakeBin, "bd-was-run")); err == nil {
		t.Fatal("the overview ran bd")
	}
	iB, iA, iO := strings.Index(doc, "`billing`"), strings.Index(doc, "`auth`"), strings.Index(doc, "`old-app`")
	if !(iB >= 0 && iB < iA && iA < iO) {
		t.Errorf("projects are not ordered by last commit:\n%s", doc)
	}
	for _, want := range []string{"last commit 2026-11-03", "1 open, 1 in progress", "provides pdf.render, invoice.send", "`b-1` P1 Fix the login redirect", "provides auth.oidc"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "Closed thing") || strings.Contains(doc, "An epic") {
		t.Errorf("closed tasks and epics must not be listed:\n%s", doc)
	}
}

func TestOverviewIsDeterministicWhateverTheIndexOrder(t *testing.T) {
	root := t.TempDir()
	a := project(t, root, "a", "2026-11-01", nil, "")
	b := project(t, root, "b", "2026-11-01", nil, "")
	c := project(t, root, "c", "2026-11-02", nil, "")
	one := Render(Load([]index.Entry{a, b, c}), DefaultOptions())
	two := Render(Load([]index.Entry{c, b, a}), DefaultOptions())
	if one != two {
		t.Errorf("order of the index changed the output:\n%s\n---\n%s", one, two)
	}
}

func TestBudgetCutsDetailBeforeProjects(t *testing.T) {
	root := t.TempDir()
	var entries []index.Entry
	for i := 0; i < 40; i++ {
		id := fmt.Sprintf("proj-%02d", i)
		entries = append(entries, project(t, root, id, fmt.Sprintf("2026-10-%02d", i%28+1), []string{"cap." + id, "other." + id}, fmt.Sprintf(tasks, id)))
	}
	ps := Load(entries)
	full := Render(ps, Options{MaxTokens: 100000})
	prev := 1 << 30
	for _, budget := range []int{2000, 800, 400, 200, 80, 30} {
		doc := Render(ps, Options{MaxTokens: budget})
		n := handoff.EstimateTokens(doc)
		if budget >= 200 && n > budget {
			t.Errorf("budget %d: %d tokens", budget, n)
		}
		if n > prev {
			t.Errorf("a smaller budget (%d) produced a longer document than a bigger one", budget)
		}
		prev = n
		if !strings.Contains(doc, "40 projects") {
			t.Errorf("the count of projects must always be there (budget %d)", budget)
		}
	}
	if !strings.Contains(full, "`proj-00`") || strings.Contains(full, "Not shown") {
		t.Errorf("with a large budget everything is shown in full")
	}
	tight := Render(ps, Options{MaxTokens: 300})
	if !strings.Contains(tight, "Not shown in full") {
		t.Errorf("at 300 tokens the least active projects should be listed by name only:\n%s", tight)
	}
}

func TestRemoteAndEmptyAndHostileInput(t *testing.T) {
	if doc := Render(nil, DefaultOptions()); !strings.Contains(doc, "index is empty") {
		t.Errorf("empty index: %s", doc)
	}
	remote := index.Entry{Source: "https://example.com/x.git", RelaiYAML: index.Manifest{ID: "x\x1b[31m", Summary: "remote one"}}
	doc := Render(Load([]index.Entry{remote}), DefaultOptions())
	if !strings.Contains(doc, "remote, not read here") || strings.Contains(doc, "\x1b") {
		t.Errorf("remote/control characters: %q", doc)
	}
}

func TestNoTaskListIsNotZeroTasks(t *testing.T) {
	root := t.TempDir()
	with := project(t, root, "tracked", "2026-11-01", nil, fmt.Sprintf(tasks, "t"))
	without := project(t, root, "untracked", "2026-11-02", nil, "")
	doc := Render(Load([]index.Entry{with, without}), DefaultOptions())
	if !strings.Contains(doc, "no task list in git") {
		t.Errorf("a project with no task export must not read as '0 open':\n%s", doc)
	}
	if strings.Count(doc, "0 open") != 0 {
		t.Errorf("unexpected '0 open':\n%s", doc)
	}
}
