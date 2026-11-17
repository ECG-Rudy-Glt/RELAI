package handoff

import (
	"os"
	"path/filepath"
	"strings"
)

// ChangeBrief is one active OpenSpec change, read from the files.
type ChangeBrief struct {
	Name  string
	Done  int
	Total int
	Next  string // first unchecked task, "" when none
}

// RepoBrief is what the workspace view shows of one repository. Everything is read from files and
// git: it never runs bd or openspec, which would be slow and rewrite state for every repository.
type RepoBrief struct {
	Branch  string
	Dirty   int
	Epics   int           // unfinished epics, not in Tasks
	Tasks   []TaskBrief   // unfinished, most urgent first; may lag the live list
	Changes []ChangeBrief // active changes that still have a task to do
	Next    []string      // lines under "### Next session" of .relai/notes.md
}

// Brief reads the compact facts of the repository at dir.
func Brief(dir string) RepoBrief {
	var b RepoBrief
	b.Branch = strings.TrimSpace(must(run(dir, "git", "--no-optional-locks", "branch", "--show-current")))
	b.Dirty = len(lines(must(run(dir, "git", "--no-optional-locks", "status", "--short"))))
	b.Tasks = SnapshotBrief(dir)
	b.Epics = SnapshotEpics(dir)
	for _, c := range specsOnDisk(dir) {
		n := nextTask(filepath.Join(dir, "openspec", "changes", c.Name, "tasks.md"))
		if n == "" && c.TotalTasks > 0 {
			continue // complete: nothing to do
		}
		b.Changes = append(b.Changes, ChangeBrief{Name: c.Name, Done: c.CompletedTasks, Total: c.TotalTasks, Next: cleanControl(n)})
	}
	if raw, err := os.ReadFile(filepath.Join(dir, ".relai", "notes.md")); err == nil {
		in := false
		for _, l := range strings.Split(string(raw), "\n") {
			t := strings.TrimSpace(l)
			switch {
			case strings.HasPrefix(t, "### "):
				in = strings.EqualFold(strings.TrimSpace(t[4:]), "Next session")
			case in && strings.HasPrefix(t, "- "):
				b.Next = append(b.Next, cleanControl(t[2:]))
			}
		}
	}
	return b
}
