package handoff

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// bigRepo has many touched files with TODO markers and a very long notes file: the shape that
// made the handoff take minutes when it cut one item at a time and re-rendered everything.
func bigRepo(t *testing.T, files, noteLines int) string {
	t.Helper()
	dir := gitRepo(t, 1)
	for i := 0; i < files; i++ {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.txt", i)), []byte("// TODO fix\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var b strings.Builder
	for i := 0; i < noteLines; i++ {
		fmt.Fprintf(&b, "note line %d\n", i)
	}
	_ = os.MkdirAll(filepath.Join(dir, ".relai"), 0o755)
	if err := os.WriteFile(filepath.Join(dir, ".relai", "notes.md"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-q", "-m", "big"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestBuild_LargeInputsStayFastAndInBudget(t *testing.T) {
	dir := bigRepo(t, 1500, 40000)
	start := time.Now()
	doc, err := Build(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if d := time.Since(start); d > 10*time.Second {
		t.Errorf("handoff took %v on a large repo", d)
	}
	if n := EstimateTokens(doc); n > 2000 {
		t.Errorf("handoff is %d tokens, budget 2000", n)
	}
}

// Cutting the notes must not corrupt a later render that shows more of them (the old code could
// not hit this because it only ever went from more lines to fewer).
func TestRenderDoesNotModifyTheNotes(t *testing.T) {
	s := sources{notes: []string{"a", "b", "c", "d"}}
	_ = render("x", s, 3, 3, 0, 0, 0, 1)
	full := render("x", s, 3, 3, 0, 0, 0, -1)
	if strings.Contains(full, "more lines") || !strings.Contains(full, "\nb\nc\nd") {
		t.Errorf("notes were altered by an earlier cut:\n%s", full)
	}
}

func TestBuild_TighterBudgetNeverShowsMoreNotes(t *testing.T) {
	dir := bigRepo(t, 20, 600)
	prev := 1 << 30
	for _, budget := range []int{2000, 1500, 1000, 700, 400, 200} {
		o := DefaultOptions()
		o.MaxTokens = budget
		doc, err := Build(dir, o)
		if err != nil {
			t.Fatal(err)
		}
		n := strings.Count(doc, "note line ")
		if n > prev {
			t.Errorf("budget %d shows %d notes, more than the larger budget before it (%d)", budget, n, prev)
		}
		prev = n
	}
}
