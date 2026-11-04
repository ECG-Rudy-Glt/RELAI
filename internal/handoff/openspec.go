package handoff

import (
	"os"
	"path/filepath"
	"strings"
)

// specsOnDisk lists the active OpenSpec changes by reading openspec/changes/*/tasks.md, for when
// the openspec command is missing or broken. Archived changes are skipped.
func specsOnDisk(dir string) []change {
	entries, err := os.ReadDir(filepath.Join(dir, "openspec", "changes"))
	if err != nil {
		return nil
	}
	var out []change
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		c := change{Name: e.Name(), Status: "from files"}
		if b, err := os.ReadFile(filepath.Join(dir, "openspec", "changes", e.Name(), "tasks.md")); err == nil {
			for _, l := range strings.Split(string(b), "\n") {
				l = strings.TrimSpace(l)
				switch {
				case strings.HasPrefix(l, "- [x] "), strings.HasPrefix(l, "- [X] "):
					c.CompletedTasks++
					c.TotalTasks++
				case strings.HasPrefix(l, "- [ ] "):
					c.TotalTasks++
				}
			}
		}
		out = append(out, c)
	}
	return out
}

// nextTask returns the first unchecked item ("- [ ] …") of an OpenSpec tasks.md, or "".
func nextTask(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); strings.HasPrefix(l, "- [ ] ") {
			return strings.TrimSpace(l[len("- [ ] "):])
		}
	}
	return ""
}
