package handoff

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", msg}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestHandoffDoesNotWriteTheGitIndex(t *testing.T) {
	dir := gitRepo(t, 1)
	// Make the stat cache stale, so a plain `git status` would refresh and rewrite .git/index.
	old := time.Now().Add(-time.Hour)
	_ = os.Chtimes(filepath.Join(dir, "f.txt"), old, old)
	_ = os.Chtimes(filepath.Join(dir, "AGENTS.md"), old, old)
	before, _ := os.ReadFile(filepath.Join(dir, ".git", "index"))
	if _, err := Build(dir, DefaultOptions()); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, ".git", "index"))
	if !bytes.Equal(before, after) {
		t.Error("relai handoff modified .git/index")
	}
}

func TestNonASCIIFileNamesAreShownAsTheyAre(t *testing.T) {
	dir := gitRepo(t, 1)
	_ = os.WriteFile(filepath.Join(dir, "café.txt"), []byte("x"), 0o644)
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "café.txt") || strings.Contains(doc, `\303`) {
		t.Errorf("file name not readable:\n%s", doc)
	}
}

func TestControlCharactersAreStripped(t *testing.T) {
	dir := gitRepo(t, 1)
	_ = os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0o644)
	commitAll(t, dir, "evil \x1b[31mred\x07 subject")
	_ = os.MkdirAll(filepath.Join(dir, ".relai"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".relai", "notes.md"), []byte("note \x1b]0;title\x07 here\n"), 0o644)
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(doc, "\x1b\x07") {
		t.Errorf("control characters in the handoff: %q", doc)
	}
	if !strings.Contains(doc, "red") || !strings.Contains(doc, "note ") {
		t.Errorf("text around the control characters was lost:\n%s", doc)
	}
}

func TestBdWithInvalidJSONIsNotReadyZero(t *testing.T) {
	fakeBD(t, `echo "this is not json"; exit 0`)
	doc, _ := Build(repoWithBeadsFiles(t, snapshotJSONL), DefaultOptions())
	if strings.Contains(doc, "Ready (unblocked, 0 total)") {
		t.Errorf("garbage from bd was shown as an empty list:\n%s", doc)
	}
	if !strings.Contains(doc, "bd returned output that could not be read") || !strings.Contains(doc, "`p-1`") {
		t.Errorf("expected the honest message and the snapshot:\n%s", doc)
	}
}

func fakeOpenspec(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "openspec"), []byte("#!/bin/sh\n"+script+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func specRepo(t *testing.T, changes int) string {
	dir := gitRepo(t, 1)
	for i := 0; i < changes; i++ {
		d := filepath.Join(dir, "openspec", "changes", fmt.Sprintf("change-%03d", i))
		_ = os.MkdirAll(d, 0o755)
		_ = os.WriteFile(filepath.Join(d, "tasks.md"), []byte("- [x] done\n- [ ] do the next thing\n"), 0o644)
	}
	_ = os.MkdirAll(filepath.Join(dir, "openspec", "changes", "archive", "old"), 0o755)
	commitAll(t, dir, "specs")
	return dir
}

func TestOpenspecMissingOrBrokenFallsBackToTheFiles(t *testing.T) {
	dir := specRepo(t, 2)
	fakeOpenspec(t, `exit 3`)
	doc, _ := Build(dir, DefaultOptions())
	for _, want := range []string{"openspec failed to list the changes", "`change-000` 1/2 tasks", "next: do the next thing"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "no OpenSpec changes in this repo") || strings.Contains(doc, "`archive`") {
		t.Errorf("wrong claim or archived change listed:\n%s", doc)
	}

	fakeOpenspec(t, `echo not-json`)
	doc, _ = Build(dir, DefaultOptions())
	if !strings.Contains(doc, "openspec returned output that could not be read") {
		t.Errorf("invalid JSON not reported:\n%s", doc)
	}
}

func TestManyChangesStayWithinTheBudget(t *testing.T) {
	dir := specRepo(t, 300)
	var js strings.Builder
	js.WriteString(`{"changes":[`)
	for i := 0; i < 300; i++ {
		if i > 0 {
			js.WriteString(",")
		}
		fmt.Fprintf(&js, `{"name":"change-%03d","completedTasks":1,"totalTasks":2,"status":"in-progress"}`, i)
	}
	js.WriteString(`]}`)
	fakeOpenspec(t, "echo '"+js.String()+"'")
	o := DefaultOptions()
	o.MaxTokens = 600
	doc, _ := Build(dir, o)
	if n := EstimateTokens(doc); n > 600 {
		t.Errorf("%d tokens with a 600 budget:\n%s", n, doc)
	}
	if !strings.Contains(doc, "more changes") {
		t.Errorf("the cut should be announced:\n%s", doc)
	}
}

func TestHandoffFromASubdirectoryIsTheRepositorys(t *testing.T) {
	dir := gitRepo(t, 1)
	sub := filepath.Join(dir, "pkg", "deep")
	_ = os.MkdirAll(sub, 0o755)
	root, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	fromSub, err := Build(sub, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if root != fromSub || strings.Contains(fromSub, "degraded") {
		t.Errorf("handoff from a subdirectory differs from the root's:\n%s", fromSub)
	}
}

func TestStateLineWordingForEmptyAndDetached(t *testing.T) {
	empty := t.TempDir()
	if out, err := exec.Command("git", "-C", empty, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	doc, err := Build(empty, DefaultOptions())
	if err != nil || !strings.Contains(doc, "no commits yet") || strings.Contains(doc, "at no commits") {
		t.Errorf("empty repo: %v\n%s", err, doc)
	}
	dir := gitRepo(t, 1)
	if out, err := exec.Command("git", "-C", dir, "checkout", "-q", "--detach").CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, out)
	}
	doc, _ = Build(dir, DefaultOptions())
	if !strings.Contains(doc, "Detached HEAD at ") || strings.Contains(doc, "Branch `?`") {
		t.Errorf("detached:\n%s", doc)
	}
}

func TestMarkersAreMarkersNotWords(t *testing.T) {
	dir := bareGitRepo(t)
	src := "// TODO: real one\n// FIXME(rudy): another\nctx := context.TODO()\nsee TODO.md for the plan\nthe todo list\n"
	_ = os.WriteFile(filepath.Join(dir, "a.go"), []byte(src), 0o644)
	commitAll(t, dir, "add")
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"TODO: real one", "FIXME(rudy): another"} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing marker %q:\n%s", want, doc)
		}
	}
	for _, noise := range []string{"context.TODO()", "see TODO.md", "the todo list"} {
		if strings.Contains(doc, noise) {
			t.Errorf("%q is not a marker:\n%s", noise, doc)
		}
	}
}
