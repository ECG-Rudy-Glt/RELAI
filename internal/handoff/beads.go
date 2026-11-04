package handoff

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// snapTask is one open task read from the JSONL export Beads keeps in git.
type snapTask struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	Type     string `json:"issue_type"`
}

// readSnapshot reads the unfinished, non-epic tasks of .beads/issues.jsonl, most urgent first. It
// is the fallback for a clone whose Beads database has not been created yet, or a machine without bd.
func readSnapshot(dir string) []snapTask {
	b, err := os.ReadFile(filepath.Join(dir, ".beads", "issues.jsonl"))
	if err != nil {
		return nil
	}
	var out []snapTask
	for _, l := range strings.Split(string(b), "\n") {
		var t snapTask
		if json.Unmarshal([]byte(l), &t) != nil || t.ID == "" {
			continue
		}
		switch t.Status {
		case "open", "in_progress", "blocked":
			if t.Type != "epic" {
				out = append(out, t)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// beadsFailure turns a failed `bd` run into a short reason, without echoing raw tool output.
func beadsFailure(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) && strings.Contains(strings.ToLower(string(ee.Stderr)), "no beads database") {
		return "this clone has no Beads database yet"
	}
	return "bd failed to read the tasks"
}

// TaskBrief is an unfinished task as other packages (the workspace view) need it.
type TaskBrief struct {
	ID, Title, Status string
	Priority          int
}

// SnapshotBrief returns the unfinished, non-epic tasks recorded in dir's .beads/issues.jsonl, most
// urgent first, without running bd. The file is a copy kept in git, so it may lag the live list.
func SnapshotBrief(dir string) []TaskBrief {
	var out []TaskBrief
	for _, t := range readSnapshot(dir) {
		out = append(out, TaskBrief{ID: t.ID, Title: cleanControl(t.Title), Status: t.Status, Priority: t.Priority})
	}
	return out
}
