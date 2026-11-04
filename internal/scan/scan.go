// Package scan finds the git repositories in a folder and the mentions between them, so a person (or an
// agent) does not have to write the list of their projects by hand, nor guess which ones belong together.
//
// Everything here is read-only and deterministic. A mention is evidence, not a verified link: the
// package reports what it saw (how many files, which kind of evidence) and never decides that two
// repositories depend on each other.
package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Repo is a git repository found in the scanned folder.
type Repo struct {
	Name       string `json:"name"` // folder name
	Path       string `json:"path"`
	ID         string `json:"id,omitempty"`   // relai.yaml id, when there is one
	Relai      bool   `json:"relai"`          // has relai.yaml
	Tracker    bool   `json:"tracker"`        // has .beads
	OpenSpec   bool   `json:"openspec"`       // has openspec/
	Agents     bool   `json:"agents"`         // has AGENTS.md or CLAUDE.md
	LastCommit string `json:"last_commit"`    // YYYY-MM-DD
	Slug       string `json:"slug,omitempty"` // owner/name of the origin remote, when it has one
}

// Find returns the git repositories directly inside dir (and dir itself if it is one), sorted by name.
// Hidden folders are skipped. It does not descend into repositories.
func Find(dir string) ([]Repo, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	var paths []string
	if isRepo(abs) {
		paths = append(paths, abs)
	}
	seen := map[string]bool{}
	if isRepo(abs) {
		if real, err := filepath.EvalSymlinks(abs); err == nil {
			seen[real] = true
		}
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		p := filepath.Join(abs, e.Name())
		// Stat, not the directory entry: a symbolic link to a repository (common when projects live
		// elsewhere and are linked into one folder) is a repository.
		if st, err := os.Stat(p); err != nil || !st.IsDir() || !isRepo(p) {
			continue
		}
		if real, err := filepath.EvalSymlinks(p); err == nil {
			if seen[real] {
				continue // the same repository reached twice
			}
			seen[real] = true
		}
		paths = append(paths, p)
	}
	var out []Repo
	for _, p := range paths {
		r := Repo{Name: filepath.Base(p), Path: p}
		r.Relai = exists(filepath.Join(p, "relai.yaml"))
		r.Tracker = exists(filepath.Join(p, ".beads"))
		r.OpenSpec = exists(filepath.Join(p, "openspec"))
		r.Agents = exists(filepath.Join(p, "AGENTS.md")) || exists(filepath.Join(p, "CLAUDE.md"))
		r.LastCommit = git(p, "log", "-1", "--format=%cs")
		r.Slug = slugOf(git(p, "remote", "get-url", "origin"))
		if r.Relai {
			if b, err := os.ReadFile(filepath.Join(p, "relai.yaml")); err == nil {
				r.ID = idFrom(b)
			}
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func isRepo(p string) bool { return exists(filepath.Join(p, ".git")) }

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func git(dir string, args ...string) string {
	args = append([]string{"--no-optional-locks"}, args...)
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

var idLine = regexp.MustCompile(`(?m)^id:\s*["']?([^"'\s#]+)`)

func idFrom(b []byte) string {
	if m := idLine.FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return ""
}

var slugRE = regexp.MustCompile(`[:/]([A-Za-z0-9._-]+/[A-Za-z0-9._-]+?)(?:\.git)?/?$`)

// slugOf turns a remote URL into "owner/name" (https, ssh and scp-like forms), or "".
func slugOf(url string) string {
	if url == "" {
		return ""
	}
	if m := slugRE.FindStringSubmatch(url); m != nil {
		return m[1]
	}
	return ""
}
