package taskref

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMentionsNeedsTheWholeID(t *testing.T) {
	cases := []struct {
		text, id string
		want     bool
	}{
		{"fix: parser (pf-1)", "pf-1", true},
		{"Closes pf-1.", "pf-1", true},                       // full stop ending the sentence
		{"see #pf-1, then pf-2", "pf-1", true},               // "#" in front, comma after
		{"pf-1\n", "pf-1", true},                             // end of a line
		{"hotfix, pf-ac3.71.1 reopened", "pf-ac3.71", false}, // a child does not name its parent
		{"epic of pf-ac3.94", "pf-ac3", false},               // nor does it name the epic above it
		{"pf-10 is another task", "pf-1", false},             // longer number
		{"xpf-1 and pf-1-b", "pf-1", false},                  // id inside a longer word
		{"tickets pf-ac3.88 to .90", "pf-ac3.88", true},      // the id itself, followed by a space
		{"done in pf-ac3.71.1.", "pf-ac3.71.1", true},        // child named, at the end of a sentence
		{"version v1.2.3 of a-1.2", "a-1", false},            // a-1.2 is a child id
		{"mentions a-1.\nnext line", "a-1", true},            // full stop then newline
		{"(a-1)", "a-1", true},                               // parentheses
		{"a-1.x", "a-1", false},                              // a dot followed by a letter continues the id
		{"see a-1: done", "a-1", true},                       // colon
		{"relai-7nk.2 in progress", "relai-7nk", false},      // Beads hierarchical id
		{"relai-7nk in progress", "relai-7nk", true},         // the parent itself
		{"no task here", "a-1", false},
	}
	for _, c := range cases {
		if got := Mentions(c.text, c.id); got != c.want {
			t.Errorf("Mentions(%q, %q) = %v, want %v", c.text, c.id, got, c.want)
		}
	}
}

// history is a real git repository whose commits each write .beads/issues.jsonl (or not) and
// return the sha, so LeftOpen is tested against git itself, not a fake.
type history struct {
	t   *testing.T
	dir string
}

func newHistory(t *testing.T) *history {
	h := &history{t: t, dir: t.TempDir()}
	h.git("init", "-q", "-b", "main")
	return h
}

func (h *history) git(args ...string) string {
	h.t.Helper()
	c := exec.Command("git", args...)
	c.Dir = h.dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := c.CombinedOutput()
	if err != nil {
		h.t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit writes jsonl as the task list ("" leaves it as it is), touches a code file and commits.
func (h *history) commit(msg, jsonl string) string {
	h.t.Helper()
	if jsonl != "" {
		_ = os.MkdirAll(filepath.Join(h.dir, ".beads"), 0o755)
		if err := os.WriteFile(filepath.Join(h.dir, ".beads", "issues.jsonl"), []byte(jsonl), 0o644); err != nil {
			h.t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(h.dir, "code.txt"), []byte(msg), 0o644)
	h.git("add", "-A")
	h.git("commit", "-q", "-m", msg)
	return h.git("rev-parse", "--short", "HEAD")
}

const (
	a1Open    = `{"id":"a-1","title":"CSV export","status":"open","updated_at":"2026-11-01T10:00:00Z"}` + "\n"
	a1Noted   = `{"id":"a-1","title":"CSV export","status":"open","updated_at":"2026-11-02T10:00:00Z","external_ref":"!17"}` + "\n"
	a1Closed  = `{"id":"a-1","title":"CSV export","status":"closed","updated_at":"2026-11-03T10:00:00Z"}` + "\n"
	a2Open    = `{"id":"a-2","title":"JSON export","status":"open","updated_at":"2026-11-01T10:00:00Z"}` + "\n"
	a2Changed = `{"id":"a-2","title":"JSON export","status":"in_progress","updated_at":"2026-11-02T11:00:00Z"}` + "\n"
)

func TestLeftOpenOnlyWhenTheCommitChangedThatTaskAndLeftItOpen(t *testing.T) {
	h := newHistory(t)
	root := h.commit("chore: plan the exports (a-1)", a1Open+a2Open)
	created := h.commit("docs: tickets a-3", a1Open+a2Open+`{"id":"a-3","title":"XML","status":"open"}`+"\n")
	updated := h.commit("chore(beads): MR !17 opened (a-1)", a1Noted+a2Open+`{"id":"a-3","title":"XML","status":"open"}`+"\n")
	reexport := h.commit("feat: CSV export (a-1)", a1Noted+a2Changed+`{"id":"a-3","title":"XML","status":"open"}`+"\n")
	untouched := h.commit("fix: CSV quoting (a-1)", "")
	closed := h.commit("feat: CSV done, Closes a-1", a1Closed+a2Changed+`{"id":"a-3","title":"XML","status":"open"}`+"\n")

	cases := []struct {
		name, sha, id string
		want          bool
	}{
		{"root commit adds the task", root, "a-1", true},
		{"commit that created a-3", created, "a-3", true},
		{"commit that edited a-1 and left it open", updated, "a-1", true},
		{"hook re-export that only changed a-2's line", reexport, "a-1", false},
		{"commit that does not touch the task list", untouched, "a-1", false},
		{"commit that closed the task", closed, "a-1", false},
		{"task absent from the list", untouched, "a-9", false},
		{"unknown commit", "0000000", "a-1", false},
	}
	for _, c := range cases {
		if got := LeftOpen(h.dir, c.sha, c.id); got != c.want {
			t.Errorf("%s: LeftOpen(%s, %s) = %v, want %v", c.name, c.sha, c.id, got, c.want)
		}
	}
}

func TestLeftOpenWritesNothing(t *testing.T) {
	h := newHistory(t)
	sha := h.commit("chore: a-1", a1Open)
	index := filepath.Join(h.dir, ".git", "index")
	before, err := os.Stat(index)
	if err != nil {
		t.Fatal(err)
	}
	_ = LeftOpen(h.dir, sha, "a-1")
	after, _ := os.Stat(index)
	if !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
		t.Error("LeftOpen changed .git/index: the handoff must stay read-only")
	}
}
