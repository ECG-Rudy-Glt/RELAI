package notes

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var today = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

const sample = `### Decisions
<!-- One line each: date — decision — why. -->
- 2026-01-10 — Go, stdlib only — single binary.
- 2026-11-10 — Recent decision.

### Pitfalls
- an undated pitfall stays where it is
- 2026-02-01 — Old pitfall
  with a second line.

### Next session
- 2026-03-01 — Merge the open PRs.
`

func write(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".relai"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, File), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestAddPutsTheEntryAtTheEndOfItsSection(t *testing.T) {
	dir := write(t, sample)
	line, err := Add(dir, Decision, "  Use  JSON\noutput ", today)
	if err != nil {
		t.Fatal(err)
	}
	if line != "- 2026-11-16 — Use JSON output" {
		t.Errorf("entry = %q", line)
	}
	got := read(t, dir, File)
	if !strings.Contains(got, "- 2026-11-10 — Recent decision.\n- 2026-11-16 — Use JSON output\n\n### Pitfalls") {
		t.Errorf("the entry is not at the end of Decisions:\n%s", got)
	}
	if _, err := Add(dir, Pitfall, "Watch the cache", today); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, File); !strings.Contains(got, "  with a second line.\n- 2026-11-16 — Watch the cache\n\n### Next session") {
		t.Errorf("the pitfall is not after the multi-line entry:\n%s", got)
	}
}

func TestAddCreatesTheFileAndTheSection(t *testing.T) {
	dir := t.TempDir()
	if _, err := Add(dir, Pitfall, "first", today); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, File); got != "### Pitfalls\n- 2026-11-16 — first\n" {
		t.Errorf("got %q", got)
	}
	if _, err := Add(dir, Decision, "second", today); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, File); got != "### Pitfalls\n- 2026-11-16 — first\n\n### Decisions\n- 2026-11-16 — second\n" {
		t.Errorf("got %q", got)
	}
	if _, err := Add(dir, Decision, "  \n ", today); err == nil {
		t.Error("an empty note must be refused")
	}
}

func TestCheckListsOldEntries(t *testing.T) {
	dir := write(t, sample)
	r, err := Check(dir, 90, "", today)
	if err != nil {
		t.Fatal(err)
	}
	if r.Dated != 4 || len(r.Old) != 3 {
		t.Fatalf("dated=%d old=%d, want 4 and 3: %+v", r.Dated, len(r.Old), r.Old)
	}
	if r.Old[1].Section != "Pitfalls" || r.Old[1].End-r.Old[1].First != 2 {
		t.Errorf("the multi-line entry is not read whole: %+v", r.Old[1])
	}
	if r, _ := Check(dir, 90, "next", today); len(r.Old) != 1 {
		t.Errorf("--section next must keep one entry, got %d", len(r.Old))
	}
	if !strings.Contains(r.Text(90, 2000), "3 entries are older than 90 days") {
		t.Errorf("report:\n%s", r.Text(90, 2000))
	}
	if _, err := Check(t.TempDir(), 90, "", today); err == nil {
		t.Error("a missing notes file must be an error")
	}
}

func TestArchiveMovesWholeEntriesAndLosesNothing(t *testing.T) {
	dir := write(t, sample)
	n, err := Archive(dir, 90, "", today)
	if err != nil || n != 3 {
		t.Fatalf("moved %d, err %v", n, err)
	}
	notes, archive := read(t, dir, File), read(t, dir, ArchiveFile)
	for _, gone := range []string{"Go, stdlib only", "Old pitfall", "with a second line", "Merge the open PRs"} {
		if strings.Contains(notes, gone) || !strings.Contains(archive, gone) {
			t.Errorf("%q should have moved to the archive\nnotes:\n%s\narchive:\n%s", gone, notes, archive)
		}
	}
	for _, kept := range []string{"### Decisions", "Recent decision", "an undated pitfall", "### Next session", "<!-- One line each"} {
		if !strings.Contains(notes, kept) {
			t.Errorf("%q should have stayed in the notes:\n%s", kept, notes)
		}
	}
	if !strings.Contains(archive, "### Pitfalls (archived 2026-11-16)") {
		t.Errorf("the archive must say where and when:\n%s", archive)
	}
	// A second run finds nothing and leaves both files as they are.
	if n, _ := Archive(dir, 90, "", today); n != 0 || read(t, dir, ArchiveFile) != archive {
		t.Error("archiving twice must change nothing")
	}
}
