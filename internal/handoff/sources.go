package handoff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/discover"
)

type task struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	Assignee  string   `json:"assignee"`
	BlockedBy []string `json:"blocked_by"`
}

type change struct {
	Name           string `json:"name"`
	CompletedTasks int    `json:"completedTasks"`
	TotalTasks     int    `json:"totalTasks"`
	Status         string `json:"status"`
	Next           string `json:"-"` // first unchecked task in tasks.md
}

type sources struct {
	branch, head string
	commits      []string
	dirty        []string
	hasBeads     bool
	beadsProblem string     // why .beads exists but its tasks cannot be read here ("" when they can)
	snapshot     []snapTask // open tasks from .beads/issues.jsonl, used when the live ones cannot be read
	ready, doing []task
	blocked      []task
	hasSpecs     bool
	changes      []change
	specsProblem string // why openspec/ exists but its changes were listed from the files instead ("" when the tool worked)
	notes        []string
	harness      []string
	degraded     bool
	recentFiles  []string
	markers      []string
	teamHint     string // "" unless at least two signals of parallel work are present
}

func collect(dir string) sources {
	var s sources
	s.branch = strings.TrimSpace(must(run(dir, "git", "branch", "--show-current")))
	s.head = strings.TrimSpace(must(run(dir, "git", "log", "-1", "--format=%h %cs")))
	s.commits = lines(must(run(dir, "git", "log", "-20", "--format=%h %cs %s")))
	s.dirty = lines(must(run(dir, "git", "status", "--short")))

	if exists(filepath.Join(dir, ".beads")) {
		if !onPath("bd") {
			s.beadsProblem = "the bd command is not installed"
		} else if out, err := run(dir, "bd", "ready", "--json"); err != nil {
			s.beadsProblem = beadsFailure(err)
		} else {
			if err := json.Unmarshal([]byte(out), &s.ready); err != nil && strings.TrimSpace(out) != "" {
				// Exit code 0 but not the JSON we asked for: say so, never show "Ready: 0".
				s.beadsProblem = "bd returned output that could not be read"
			} else {
				s.hasBeads = true
				_ = json.Unmarshal([]byte(must(run(dir, "bd", "list", "--status", "in_progress", "--json"))), &s.doing)
				_ = json.Unmarshal([]byte(must(run(dir, "bd", "blocked", "--json"))), &s.blocked)
			}
		}
		if s.beadsProblem != "" {
			s.snapshot = readSnapshot(dir)
		}
	}
	if exists(filepath.Join(dir, "openspec")) {
		var out struct {
			Changes []change `json:"changes"`
		}
		switch {
		case !onPath("openspec"):
			s.specsProblem = "the openspec command is not installed"
		default:
			raw, err := run(dir, "openspec", "list", "--json")
			if err != nil {
				s.specsProblem = "openspec failed to list the changes"
			} else if jerr := json.Unmarshal([]byte(raw), &out); jerr != nil {
				s.specsProblem = "openspec returned output that could not be read"
			} else {
				s.hasSpecs = true
			}
		}
		if s.specsProblem != "" {
			out.Changes = specsOnDisk(dir) // same facts, read from the files, without the tool
		}
		s.changes = out.Changes
		for i, c := range s.changes {
			s.changes[i].Next = nextTask(filepath.Join(dir, "openspec", "changes", c.Name, "tasks.md"))
		}
	}
	s.teamHint = teamHint(ownerCount(dir), len(dedupe(lines(must(run(dir, "git", "log", "--since=14.days", "--format=%an"))))), assigneeCount(s.doing))
	inv := discover.Scan(dir)
	s.harness = inv.Lines(8)
	s.degraded = !inv.HasCoreHarness()
	if s.degraded {
		s.recentFiles = dedupe(lines(must(run(dir, "git", "log", "--since=7.days", "--name-only", "--pretty=format:"))))
		s.markers = lines(must(run(dir, "git", "grep", "-n", "-I", "-E", markerPattern)))
	}
	if b, err := os.ReadFile(filepath.Join(dir, ".relai", "notes.md")); err == nil {
		s.notes = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
	return s
}

// markerPattern matches a TODO or FIXME written as a marker ("TODO:", "TODO(name)", "FIXME:"), not
// the word in prose, a file named TODO.md, or Go's context.TODO() (a '.' or a letter before it).
const markerPattern = `(^|[^.A-Za-z_])(TODO|FIXME)[:(]`
