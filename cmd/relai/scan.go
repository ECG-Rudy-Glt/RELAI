package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/scan"
)

// scanSources finds the git repositories of every folder in dirs (depth levels down) and returns their
// paths, in the order found. A repository reached more than once, through two folders that overlap or
// through a symbolic link, is returned once, by the first folder that found it. One "scan:" line per
// folder is written to w; with several folders, the count is of the repositories that folder added.
func scanSources(dirs []string, depth int, w io.Writer) ([]string, error) {
	var sources []string
	seen := map[string]bool{}
	for _, dir := range dirs {
		repos, err := scan.FindDepth(dir, depth)
		if err != nil {
			return nil, err
		}
		var kept []scan.Repo
		already := 0
		for _, r := range repos {
			key := scan.Identity(r.Path) // a symbolic link or a worktree of a repository already found is that repository
			if seen[key] {
				already++
				continue
			}
			seen[key] = true
			kept = append(kept, r)
		}
		var without []string
		for _, r := range kept {
			sources = append(sources, r.Path)
			if !r.Relai {
				without = append(without, r.Name)
			}
		}
		fmt.Fprintf(w, "scan: %d repositories in %s, %d with a relai.yaml", len(kept), dir, len(kept)-len(without))
		if already > 0 {
			fmt.Fprintf(w, "; %d already counted from a previous folder", already)
		}
		if len(without) > 0 {
			fmt.Fprintf(w, "; without one (skipped, run `relai init` there to include them): %s", strings.Join(without, ", "))
		}
		fmt.Fprintln(w)
	}
	return sources, nil
}
