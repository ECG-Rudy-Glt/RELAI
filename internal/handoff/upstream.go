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
	ID      string `json:"id"`
	Commits int    `json:"commits"`
	Last    string `json:"last_commit"` // YYYY-MM-DD of the linked repository's last commit
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
		out = append(out, Moved{ID: cleanControl(l), Commits: n, Last: strings.TrimSpace(string(last))})
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
