package workspace

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
)

func commitOn(t *testing.T, dir, date, msg string) {
	t.Helper()
	_ = os.WriteFile(filepath.Join(dir, msg), []byte(msg), 0o644)
	env := append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		"GIT_AUTHOR_DATE="+date+"T12:00:00Z", "GIT_COMMITTER_DATE="+date+"T12:00:00Z")
	for _, a := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", msg}} {
		c := exec.Command("git", a...)
		c.Dir, c.Env = dir, env
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", a, err, out)
		}
	}
}

func TestHandoffBlocksAndUpstream(t *testing.T) {
	root := t.TempDir()
	app := project(t, root, "app", "2026-11-01", nil, fmt.Sprintf(tasks, "a")).Source
	lib := project(t, root, "lib", "2026-10-01", nil, "").Source
	_ = os.WriteFile(filepath.Join(app, "relai.yaml"), []byte("id: app\nlinks: [lib, nowhere]\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(app, ".relai"), 0o755)
	_ = os.WriteFile(filepath.Join(app, ".relai", "notes.md"), []byte("### Decisions\n- d\n\n### Next session\n- finish the login\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(app, "openspec", "changes", "login"), 0o755)
	_ = os.WriteFile(filepath.Join(app, "openspec", "changes", "login", "tasks.md"), []byte("- [x] a\n- [ ] write the form\n"), 0o644)
	commitOn(t, app, "2026-11-01", "app2")
	commitOn(t, lib, "2026-11-05", "lib2") // lib moved after app's last commit
	commitOn(t, lib, "2026-11-06", "lib3")

	fakeBin := t.TempDir()
	_ = os.WriteFile(filepath.Join(fakeBin, "bd"), []byte("#!/bin/sh\ntouch "+filepath.Join(fakeBin, "bd-was-run")+"\nexit 1\n"), 0o755)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))

	bs, err := LoadBlocks(root, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(fakeBin, "bd-was-run")); err == nil {
		t.Fatal("the workspace handoff ran bd")
	}
	doc := RenderHandoff(bs, Options{MaxTokens: 1500})
	for _, want := range []string{
		"## app", "1 open, 1 in progress (+1 epic not counted)",
		"upstream changes: `lib` has 2 commit(s)",
		"change `login` 1/2 — next: write the form", "next session: finish the login",
		"## lib", "no task list in git",
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
	if strings.Contains(doc, "nowhere") {
		t.Error("a link to a repository that is not in the folder must not be reported as upstream")
	}
	if bs[0].ID != "lib" {
		t.Error("most recently active first")
	}
}

func TestHandoffBudgetNeverExceeded(t *testing.T) {
	var bs []Block
	for i := 0; i < 40; i++ {
		bs = append(bs, Block{Name: fmt.Sprint("r", i), ID: fmt.Sprint("repo-", i), LastCommit: "2026-11-01", Open: 9,
			Ready: []handoff.TaskBrief{{ID: "x-1", Title: strings.Repeat("long ", 30), Priority: 1}},
			Next:  []string{strings.Repeat("next ", 40)}})
	}
	for _, max := range []int{300, 800, 1500} {
		doc := RenderHandoff(bs, Options{MaxTokens: max})
		if got := handoff.EstimateTokens(doc); got > max {
			t.Errorf("budget %d: got %d tokens", max, got)
		}
	}
}
