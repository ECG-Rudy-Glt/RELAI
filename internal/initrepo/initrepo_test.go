package initrepo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitIsIdempotentAndNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	r, err := Init(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Created) != 3 {
		t.Fatalf("want 3 files created, got %v", r.Created)
	}
	custom := []byte("# my own notes\n")
	notes := filepath.Join(dir, ".relai", "notes.md")
	_ = os.WriteFile(notes, custom, 0o644)

	r, err = Init(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Created) != 0 || len(r.Skipped) != 3 {
		t.Fatalf("second run must only skip, got %+v", r)
	}
	if got, _ := os.ReadFile(notes); string(got) != string(custom) {
		t.Fatal("existing notes.md was overwritten")
	}
}

func TestAgentsTemplateCoversNoCLIAndCrossRepoCheck(t *testing.T) {
	dir := t.TempDir()
	if _, err := Init(dir, true); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"not installed or not found", "relai find <capability>"} {
		if !strings.Contains(got, want) {
			t.Errorf("AGENTS.md template missing %q:\n%s", want, got)
		}
	}
}

func TestInitLinksExistingClaudeMd(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# rules\n"), 0o644)
	if _, err := Init(dir, true); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(dir, "AGENTS.md"))
	if err != nil || target != "CLAUDE.md" {
		t.Fatalf("AGENTS.md should link to CLAUDE.md, got %q (%v)", target, err)
	}
}

func TestClaudeCodeWiringCreatesImportAndHook(t *testing.T) {
	dir := t.TempDir()
	r, err := InitWith(dir, Options{LinkClaude: true, ClaudeCode: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Created) != 6 {
		t.Fatalf("created = %v, want AGENTS.md, relai.yaml, notes.md, CLAUDE.md, hook, settings", r.Created)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md")); string(b) != "@AGENTS.md\n" {
		t.Errorf("CLAUDE.md = %q, want the import line", b)
	}
	hook := filepath.Join(dir, ".claude", "hooks", "session-start.sh")
	fi, err := os.Stat(hook)
	if err != nil || fi.Mode()&0o111 == 0 {
		t.Fatalf("hook missing or not executable: %v %v", err, fi)
	}
	settings, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	var parsed map[string]any
	if err := json.Unmarshal(settings, &parsed); err != nil {
		t.Fatalf("settings.json is not valid JSON: %v\n%s", err, settings)
	}
	if !strings.Contains(string(settings), "SessionStart") || !strings.Contains(string(settings), ".claude/hooks/session-start.sh") {
		t.Errorf("settings do not register the hook:\n%s", settings)
	}
}

func TestClaudeCodeWiringOffByDefaultForInit(t *testing.T) {
	dir := t.TempDir()
	if _, err := Init(dir, true); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"CLAUDE.md", ".claude"} {
		if _, err := os.Lstat(filepath.Join(dir, f)); err == nil {
			t.Errorf("%s created by plain Init", f)
		}
	}
}

func TestClaudeCodeWiringNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, ".claude"), 0o755)
	mine := `{"permissions":{"allow":["Bash(ls)"]}}`
	_ = os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), []byte(mine), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# my rules\n"), 0o644)
	r, err := InitWith(dir, Options{LinkClaude: true, ClaudeCode: true})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")); string(b) != mine {
		t.Errorf("existing settings.json was modified: %s", b)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "CLAUDE.md")); string(b) != "# my rules\n" {
		t.Errorf("existing CLAUDE.md was modified: %s", b)
	}
	joined := strings.Join(r.Skipped, "\n")
	if !strings.Contains(joined, "settings.json") || !strings.Contains(joined, "CLAUDE.md") || !strings.Contains(joined, "SessionStart") {
		t.Errorf("the user is not told what to do by hand:\n%s", joined)
	}
	// the second run changes nothing at all
	r2, _ := InitWith(dir, Options{LinkClaude: true, ClaudeCode: true})
	if len(r2.Created) != 0 {
		t.Errorf("second run created %v", r2.Created)
	}
}

func TestHookScriptNeverFailsAndFallsBackToPlainFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := InitWith(dir, Options{ClaudeCode: true}); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, ".relai", "notes.md"), []byte("### Decisions\n- we chose X\n"), 0o644)
	// run the hook with a PATH that has no relai at all
	cmd := exec.Command("/bin/sh", filepath.Join(dir, ".claude", "hooks", "session-start.sh"))
	cmd.Env = []string{"PATH=/usr/bin:/bin", "CLAUDE_PROJECT_DIR=" + dir}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("hook failed: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "relai CLI is not installed") || !strings.Contains(string(out), "we chose X") {
		t.Errorf("fallback did not print the plain files:\n%s", out)
	}
	// and with a project dir that does not exist it still exits 0
	cmd = exec.Command("/bin/sh", filepath.Join(dir, ".claude", "hooks", "session-start.sh"))
	cmd.Env = []string{"PATH=/usr/bin:/bin", "CLAUDE_PROJECT_DIR=/no/such/dir"}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("hook must never fail, got %v\n%s", err, out)
	}
}

func TestSlug(t *testing.T) {
	for in, want := range map[string]string{"My Widget-x": "my-widget-x", "日本語": "project", "": "project", "a_b.c": "a-b-c"} {
		if got := Slug(in); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestInitIsAllOrNothingWhenRelaiIsAFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".relai"), []byte("in the way"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := InitWith(dir, Options{ClaudeCode: true}); err == nil {
		t.Fatal("expected an error")
	}
	for _, f := range []string{"AGENTS.md", "relai.yaml", "CLAUDE.md", ".claude"} {
		if _, err := os.Lstat(filepath.Join(dir, f)); err == nil {
			t.Errorf("%s was created before the error", f)
		}
	}
}

func TestClaudeSettingsHookHasATimeout(t *testing.T) {
	if !strings.Contains(ClaudeSettings, `"timeout": 30`) {
		t.Error("the generated SessionStart hook needs a timeout so a slow tool cannot block a session start")
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()
	if out, err := exec.Command("git", "-C", dir, "init", "-q", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
}

func TestGitHookIsInstalledExecutableAndNeverOverwritten(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)
	r, err := InitWith(dir, Options{GitHook: true})
	if err != nil {
		t.Fatal(err)
	}
	hook := filepath.Join(dir, ".git", "hooks", "pre-commit")
	st, err := os.Stat(hook)
	if err != nil || st.Mode().Perm()&0o111 == 0 {
		t.Fatalf("hook missing or not executable: %v", err)
	}
	if !strings.Contains(strings.Join(r.Created, " "), "pre-commit") {
		t.Errorf("not reported: %v", r.Created)
	}
	_ = os.WriteFile(hook, []byte("#!/bin/sh\n# mine\n"), 0o755)
	r, _ = InitWith(dir, Options{GitHook: true})
	if b, _ := os.ReadFile(hook); !strings.Contains(string(b), "# mine") {
		t.Error("an existing hook was overwritten")
	}
	if !strings.Contains(strings.Join(r.Skipped, " "), "pre-commit") {
		t.Errorf("the skip is not reported: %v", r.Skipped)
	}
}

func TestGitHookRefreshesTheExportInTheSameCommitAndNeverBlocks(t *testing.T) {
	dir := t.TempDir()
	gitInit(t, dir)
	if _, err := InitWith(dir, Options{GitHook: true}); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(dir, ".beads"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".beads", "last-touched"), []byte("x-1"), 0o644)
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls")
	// a fake bd: "export -o FILE" writes one task with an owner e-mail
	_ = os.WriteFile(filepath.Join(bin, "bd"), []byte("#!/bin/sh\necho x >> "+calls+"\nout=\"$3\"\necho '{\"id\":\"x-1\",\"owner\":\"me@example.com\",\"title\":\"t\"}' > \"$out\"\n"), 0o755)
	env := append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	commit := func(msg string) {
		t.Helper()
		for _, a := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", msg}} {
			c := exec.Command("git", a...)
			c.Dir, c.Env = dir, env
			if out, err := c.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", a, err, out)
			}
		}
	}
	commit("first")
	out, _ := exec.Command("git", "-C", dir, "show", "--stat", "--format=", "HEAD").Output()
	if !strings.Contains(string(out), ".beads/issues.jsonl") {
		t.Fatalf("the export is not in the commit that changed the tasks:\n%s", out)
	}
	b, _ := os.ReadFile(filepath.Join(dir, ".beads", "issues.jsonl"))
	if strings.Contains(string(b), "owner") || !strings.Contains(string(b), `"id":"x-1"`) {
		t.Errorf("export content wrong (the owner e-mail must be removed): %s", b)
	}
	// nothing touched bd since: the next commit must not ask it again
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644)
	commit("second")
	if c, _ := os.ReadFile(calls); strings.Count(string(c), "x") != 1 {
		t.Errorf("bd was asked again although nothing changed: %q", c)
	}
	// a failing bd must not block the commit
	_ = os.WriteFile(filepath.Join(bin, "bd"), []byte("#!/bin/sh\nexit 1\n"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".beads", "last-touched"), []byte("x-2"), 0o644)
	future := time.Now().Add(time.Minute)
	_ = os.Chtimes(filepath.Join(dir, ".beads", "last-touched"), future, future)
	commit("third, bd fails")
}
