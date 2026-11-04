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
	// Worktrees are the other working copies of this same repository found in the scanned folder
	// (git worktrees). They are one repository, not several.
	Worktrees []string `json:"worktrees,omitempty"`
}

// Find returns the git repositories directly inside dir (and dir itself if it is one), sorted by name.
// It is FindDepth(dir, 1).
func Find(dir string) ([]Repo, error) { return FindDepth(dir, 1) }

// skipDirs are folders that never hold the projects of a person, whatever their depth.
var skipDirs = map[string]bool{"node_modules": true, "vendor": true, "graphify-out": true, "dist": true, "build": true, "target": true, "__pycache__": true}

// FindDepth returns the git repositories inside dir, looking depth folder levels down (1 = directly
// inside; 2 = also inside the folders that are not repositories themselves, and so on). Symbolic links
// to repositories count and a repository reached twice is listed once. It never descends into a
// repository, nor into hidden or generated folders. Repos found below the first level are named by their
// path relative to dir ("clients/acme") so two projects with the same folder name stay distinct.
func FindDepth(dir string, depth int) ([]Repo, error) {
	if depth < 1 {
		depth = 1
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if _, err := os.ReadDir(abs); err != nil {
		return nil, err
	}
	var paths []string
	seen := map[string]bool{}
	mark := func(p string) bool {
		if real, err := filepath.EvalSymlinks(p); err == nil {
			if seen[real] {
				return false
			}
			seen[real] = true
		}
		return true
	}
	var walk func(d string, level int)
	walk = func(d string, level int) {
		entries, err := os.ReadDir(d)
		if err != nil {
			return
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") || skipDirs[e.Name()] {
				continue
			}
			p := filepath.Join(d, e.Name())
			// Stat, not the directory entry: a symbolic link to a repository (common when projects live
			// elsewhere and are linked into one folder) is a repository.
			st, err := os.Stat(p)
			if err != nil || !st.IsDir() {
				continue
			}
			if isRepo(p) {
				if mark(p) {
					paths = append(paths, p)
				}
				continue
			}
			if level < depth && mark(p) {
				walk(p, level+1)
			}
		}
	}
	if isRepo(abs) {
		mark(abs)
		paths = append(paths, abs)
	} else {
		walk(abs, 1)
	}
	paths, wt := mergeWorktrees(paths)
	var out []Repo
	for _, p := range paths {
		name := filepath.Base(p)
		if rel, err := filepath.Rel(abs, p); err == nil && rel != "." && strings.Contains(rel, string(filepath.Separator)) {
			name = filepath.ToSlash(rel)
		}
		r := Repo{Name: name, Path: p, Worktrees: wt[p]}
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

// mergeWorktrees keeps one path per repository: the working copies sharing a git directory (git
// worktrees) are one project. The main working copy wins (its .git is a folder); with only linked
// copies the first path in name order stays. The others are returned under the kept path.
func mergeWorktrees(paths []string) ([]string, map[string][]string) {
	group := map[string][]string{}
	var order []string
	for _, p := range paths {
		key := git(p, "rev-parse", "--path-format=absolute", "--git-common-dir")
		if key == "" {
			key = p
		}
		if _, ok := group[key]; !ok {
			order = append(order, key)
		}
		group[key] = append(group[key], p)
	}
	var kept []string
	extra := map[string][]string{}
	for _, k := range order {
		ps := group[k]
		sort.Strings(ps)
		main := ps[0]
		for _, p := range ps {
			if st, err := os.Stat(filepath.Join(p, ".git")); err == nil && st.IsDir() {
				main = p
				break
			}
		}
		kept = append(kept, main)
		for _, p := range ps {
			if p != main {
				extra[main] = append(extra[main], p)
			}
		}
	}
	return kept, extra
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
