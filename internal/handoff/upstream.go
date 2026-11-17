package handoff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
	"github.com/ECG-Rudy-Glt/RELAI/internal/scan"
)

// Moved says that a repository this one links to has commits newer than this one's last commit.
type Moved struct {
	ID       string   `json:"id"`
	Commits  int      `json:"commits"`
	Last     string   `json:"last_commit"`        // YYYY-MM-DD of the linked repository's last commit
	Subjects []string `json:"subjects,omitempty"` // subjects of its newest commits since, at most 2: what changed, not only that something did
	Notes    []string `json:"notes,omitempty"`    // its dated notes (decisions, pitfalls) since, at most 2: the decision itself, not a pointer to it
	Path     string   `json:"-"`                  // where it was found, for the command to read it
}

// datedNotesSince returns the lines of path's .relai/notes.md that start with "- YYYY-MM-DD" on or after day,
// outside the "Next session" section, newest first, at most n.
func datedNotesSince(path, day string, n int) []string {
	b, err := os.ReadFile(filepath.Join(path, ".relai", "notes.md"))
	if err != nil {
		return nil
	}
	var out []string
	next := false
	for _, l := range strings.Split(string(b), "\n") {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "### ") {
			next = strings.EqualFold(strings.TrimSpace(t[4:]), "Next session")
			continue
		}
		if next || !strings.HasPrefix(t, "- ") || len(t) < 12 {
			continue
		}
		d := t[2:12]
		if len(d) == 10 && d[4] == '-' && d[7] == '-' && d >= day {
			out = append(out, short(cleanControl(t[2:]), 140))
		}
	}
	// notes are appended in date order: the newest are at the end
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// UpstreamMoved lists, for each id of links that find resolves to a repository, the commits made there
// after the last commit of the repository at dir: what moved while nobody touched this one. A link
// that find does not resolve is left out, never guessed. Reads git only.
func UpstreamMoved(dir string, links []string, find func(id string) string) []Moved {
	since, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%cI").Output()
	if err != nil || strings.TrimSpace(string(since)) == "" {
		return nil
	}
	me := scan.Identity(dir)
	var out []Moved
	for _, l := range links {
		path := find(l)
		if path == "" || scan.Identity(path) == me {
			continue
		}
		o, err := exec.Command("git", "-C", path, "rev-list", "--count", "--since="+strings.TrimSpace(string(since)), "HEAD").Output()
		if err != nil {
			continue
		}
		n, _ := strconv.Atoi(strings.TrimSpace(string(o)))
		if n == 0 {
			continue
		}
		last, _ := exec.Command("git", "-C", path, "log", "-1", "--format=%cs").Output()
		var subjects []string
		if so, err := exec.Command("git", "-C", path, "log", "-2", "--format=%s", "--since="+strings.TrimSpace(string(since))).Output(); err == nil {
			for _, l := range lines(string(so)) {
				subjects = append(subjects, short(cleanControl(l), 90))
			}
		}
		day := strings.TrimSpace(string(since))
		if len(day) >= 10 {
			day = day[:10]
		}
		out = append(out, Moved{ID: cleanControl(l), Commits: n, Last: strings.TrimSpace(string(last)), Subjects: subjects,
			Notes: datedNotesSince(path, day, 2), Path: path})
	}
	return out
}

// siblingUpstream finds the linked repositories next to dir (same parent folder, by relai.yaml id or
// folder name) and returns what moved in them.
func siblingUpstream(dir string) []Moved {
	b, err := os.ReadFile(filepath.Join(dir, "relai.yaml"))
	if err != nil {
		return nil
	}
	links := index.ParseManifest(b).Links
	if len(links) == 0 {
		return nil
	}
	repos, err := scan.FindDepth(filepath.Dir(dir), 1)
	if err != nil {
		return nil
	}
	byID := map[string]string{}
	for _, r := range repos {
		byID[r.Name] = r.Path
		if r.ID != "" {
			byID[r.ID] = r.Path
		}
	}
	return UpstreamMoved(dir, links, func(id string) string { return byID[id] })
}
