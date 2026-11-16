package handoff

import (
	"encoding/json"
	"strconv"
)

// JSONTask is a task in the JSON form of the handoff, whatever backend it comes from.
type JSONTask struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  *int     `json:"priority,omitempty"`
	Assignees []string `json:"assignees,omitempty"`
	Labels    []string `json:"labels,omitempty"`
	BlockedBy []string `json:"blocked_by,omitempty"`
}

// JSONChange is an active spec change.
type JSONChange struct {
	Name      string `json:"name"`
	Completed int    `json:"completed_tasks"`
	Total     int    `json:"total_tasks"`
	Status    string `json:"status"`
	Next      string `json:"next,omitempty"`
}

// JSONDoc is the handoff as data (spec/handoff.md §9): the same facts as the Markdown, without the
// budget cuts, for tools that would otherwise have to parse Markdown.
type JSONDoc struct {
	Format   string `json:"format"`
	Repo     string `json:"repo"`
	Degraded bool   `json:"degraded"`
	State    struct {
		Branch      string   `json:"branch"`
		Head        string   `json:"head"`
		Uncommitted []string `json:"uncommitted"`
		Commits     []string `json:"commits"`
	} `json:"state"`
	Tasks struct {
		// Source is "beads", "beads-snapshot" (read from .beads/issues.jsonl, may lag), "github" or "none".
		Source     string     `json:"source"`
		Problem    string     `json:"problem,omitempty"`
		InProgress []JSONTask `json:"in_progress"`
		Ready      []JSONTask `json:"ready"`
		Blocked    []JSONTask `json:"blocked"`
		// Open holds the tasks of a backend that does not say which are ready (snapshot, GitHub).
		Open []JSONTask `json:"open"`
	} `json:"tasks"`
	Changes        []JSONChange `json:"changes"`
	ChangesProblem string       `json:"changes_problem,omitempty"`
	Harness        []string     `json:"harness"`
	Notes          []string     `json:"notes"`
	RecentFiles    []string     `json:"recent_files,omitempty"`
	Markers        []string     `json:"markers,omitempty"`
}

// BuildJSON returns the handoff of the repository at dir as indented JSON.
func BuildJSON(dir string) ([]byte, error) {
	root, err := repoRoot(dir)
	if err != nil {
		return nil, err
	}
	s := collect(root)
	var d JSONDoc
	d.Format, d.Repo, d.Degraded = FormatVersion, repoName(root), s.degraded
	d.State.Branch, d.State.Head = s.branch, s.head
	d.State.Uncommitted = cleanAll(stripStatus(s.dirty))
	d.State.Commits = cleanAll(s.commits)

	conv := func(ts []task) []JSONTask {
		out := []JSONTask{}
		for _, t := range ts {
			p := t.Priority
			j := JSONTask{ID: t.ID, Title: cleanControl(t.Title), Priority: &p, BlockedBy: t.BlockedBy}
			if t.Assignee != "" {
				j.Assignees = []string{t.Assignee}
			}
			out = append(out, j)
		}
		return out
	}
	d.Tasks.Source = "none"
	d.Tasks.InProgress, d.Tasks.Ready, d.Tasks.Blocked, d.Tasks.Open = conv(s.doing), conv(s.ready), conv(s.blocked), []JSONTask{}
	switch {
	case s.hasBeads:
		d.Tasks.Source = "beads"
	case s.beadsProblem != "":
		d.Tasks.Source, d.Tasks.Problem = "beads-snapshot", s.beadsProblem
		for _, t := range s.snapshot {
			p := t.Priority
			j := JSONTask{ID: t.ID, Title: cleanControl(t.Title), Priority: &p}
			if t.Status == "in_progress" {
				d.Tasks.InProgress = append(d.Tasks.InProgress, j)
			} else {
				d.Tasks.Open = append(d.Tasks.Open, j)
			}
		}
	case s.hasGitHub:
		d.Tasks.Source = "github"
		for _, i := range s.ghIssues {
			j := JSONTask{ID: "#" + strconv.Itoa(i.Number), Title: cleanControl(i.Title), Assignees: i.assignees(), Labels: i.labels()}
			if len(j.Assignees) > 0 {
				d.Tasks.InProgress = append(d.Tasks.InProgress, j)
			} else {
				d.Tasks.Open = append(d.Tasks.Open, j)
			}
		}
	default:
		d.Tasks.Problem = s.ghProblem
	}
	d.Changes = []JSONChange{}
	for _, c := range s.changes {
		d.Changes = append(d.Changes, JSONChange{Name: c.Name, Completed: c.CompletedTasks, Total: c.TotalTasks, Status: c.Status, Next: cleanControl(c.Next)})
	}
	d.ChangesProblem = s.specsProblem
	d.Harness = cleanAll(s.harness)
	d.Notes = cleanAll(s.notes)
	d.RecentFiles, d.Markers = cleanAll(s.recentFiles), cleanAll(s.markers)
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// stripStatus removes the two status letters `git status --short` puts before each path.
func stripStatus(dirty []string) []string {
	out := make([]string, 0, len(dirty))
	for _, d := range dirty {
		if len(d) > 3 {
			d = d[3:]
		}
		out = append(out, d)
	}
	return out
}

func cleanAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, cleanControl(x))
	}
	return out
}
