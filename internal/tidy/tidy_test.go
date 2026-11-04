package tidy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func sh(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return string(out)
}

const export = `{"_type":"issue","id":"p-1","title":"One","status":"open","priority":1,"issue_type":"task"}
{"_type":"issue","id":"p-2","title":"Two","status":"in_progress","priority":1,"issue_type":"task"}
{"_type":"issue","id":"p-3","title":"Three","status":"open","priority":2,"issue_type":"task"}
{"_type":"issue","id":"p-4","title":"Four","status":"open","priority":2,"issue_type":"task"}
{"_type":"issue","id":"p-10","title":"Ten","status":"open","priority":2,"issue_type":"task"}
{"_type":"issue","id":"p-9","title":"Done already","status":"closed","priority":2,"issue_type":"task"}
`

func repo(t *testing.T) string {
	dir := t.TempDir()
	sh(t, dir, "init", "-q", "-b", "main")
	_ = os.MkdirAll(filepath.Join(dir, ".beads"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".beads", "issues.jsonl"), []byte(export), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "a"), []byte("a"), 0o644)
	sh(t, dir, "add", "-A")
	sh(t, dir, "commit", "-q", "-m", "init")
	commit := func(msg string) {
		_ = os.WriteFile(filepath.Join(dir, "f"), []byte(msg), 0o644)
		sh(t, dir, "add", "-A")
		sh(t, dir, "commit", "-q", "-m", msg)
	}
	commit("fix: the parser\n\nCloses p-1")
	commit("feat: more work on p-2 (not finished)")
	commit("fixes #p-3 and resolves p-4: done")
	commit("chore: p-10 is a different task from p-1, mention only")
	commit("docs: Closes p-9") // already closed: must be ignored
	return dir
}

func ids(cs []Closure) string {
	var s []string
	for _, c := range cs {
		s = append(s, c.Task)
	}
	return strings.Join(s, ",")
}

func TestOnlyAnExplicitKeywordClosesATask(t *testing.T) {
	dir := repo(t)
	p, err := Build(dir, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(ids(p.Close), ",")
	want := map[string]bool{"p-1": true, "p-3": true, "p-4": true}
	if len(got) != 3 {
		t.Fatalf("close = %v, want p-1, p-3 and p-4", got)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("closed %s, which no commit says it closes", g)
		}
	}
	if strings.Contains(ids(p.Close), "p-10") || strings.Contains(ids(p.Close), "p-2") || strings.Contains(ids(p.Close), "p-9") {
		t.Errorf("close = %v: p-10 (prefix of p-1's id space), p-2 (only named) and p-9 (already closed) must not be closed", ids(p.Close))
	}
	if !strings.Contains(ids(p.Named), "p-2") || !strings.Contains(ids(p.Named), "p-10") {
		t.Errorf("tasks named without a keyword should be reported, got %v", ids(p.Named))
	}
}

func TestNothingIsClosedFromAFeatureBranch(t *testing.T) {
	dir := repo(t)
	sh(t, dir, "checkout", "-q", "-b", "feat/x")
	_ = os.WriteFile(filepath.Join(dir, "g"), []byte("g"), 0o644)
	sh(t, dir, "add", "-A")
	sh(t, dir, "commit", "-q", "-m", "wip\n\nCloses p-3")
	p, _ := Build(dir, 0)
	if p.OnDefault || len(p.Close) != 0 {
		t.Errorf("on a feature branch the work is not merged: OnDefault=%v close=%v", p.OnDefault, p.Close)
	}
}

func TestApplyClosesWithBdRewritesTheExportWithoutOwnerAndDeletesOnlyMergedBranches(t *testing.T) {
	dir := repo(t)
	sh(t, dir, "branch", "merged-one") // points at main: fully merged
	sh(t, dir, "checkout", "-q", "-b", "unmerged")
	_ = os.WriteFile(filepath.Join(dir, "u"), []byte("u"), 0o644)
	sh(t, dir, "add", "-A")
	sh(t, dir, "commit", "-q", "-m", "not merged")
	sh(t, dir, "checkout", "-q", "main")

	bin := t.TempDir()
	log := filepath.Join(bin, "log")
	_ = os.WriteFile(filepath.Join(bin, "bd"), []byte("#!/bin/sh\necho \"$@\" >> "+log+"\nif [ \"$1\" = export ]; then echo '{\"_type\":\"issue\",\"id\":\"p-2\",\"owner\":\"me@example.com\",\"title\":\"Two\",\"status\":\"open\"}' > \"$3\"; fi\n"), 0o755)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	p, _ := Build(dir, 0)
	if strings.Join(p.Branches, ",") != "merged-one" {
		t.Fatalf("merged branches = %v (the unmerged one and main must stay)", p.Branches)
	}
	r := p.Apply(dir)
	if len(r.Closed) != 3 || !r.ExportRewritten || len(r.DeletedBranches) != 1 {
		t.Fatalf("result = %+v", r)
	}
	b, _ := os.ReadFile(log)
	for _, want := range []string{"close p-1 --reason closed by commit", "close p-3", "close p-4", "export -o"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("bd was not called with %q:\n%s", want, b)
		}
	}
	ex, _ := os.ReadFile(filepath.Join(dir, ".beads", "issues.jsonl"))
	if strings.Contains(string(ex), "owner") || !strings.Contains(string(ex), `"id":"p-2"`) {
		t.Errorf("export = %s: the owner e-mail must be removed", ex)
	}
	if out := sh(t, dir, "branch", "--list", "unmerged"); !strings.Contains(out, "unmerged") {
		t.Error("an unmerged branch was deleted")
	}
	if st := sh(t, dir, "log", "-1", "--format=%s"); strings.Contains(st, "tidy") {
		t.Error("tidy must not commit")
	}
}

func TestApplyWithoutBdLeavesTasksOpenAndSaysSo(t *testing.T) {
	dir := repo(t)
	t.Setenv("PATH", t.TempDir()+string(os.PathListSeparator)+filepath.Dir(mustLook("git")))
	p, _ := Build(dir, 0)
	r := p.Apply(dir)
	if len(r.Closed) != 0 || len(r.Notes) == 0 || !strings.Contains(r.Notes[0], "bd is not installed") {
		t.Errorf("result = %+v", r)
	}
}

func mustLook(n string) string {
	p, err := exec.LookPath(n)
	if err != nil {
		panic(err)
	}
	return p
}
