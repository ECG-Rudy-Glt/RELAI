package initrepo

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
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
