package handoff

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

// TeamURL is where the optional team layer lives.
const TeamURL = "https://github.com/ECG-Rudy-Glt/RELAI-TEAM"

// teamHint returns the one-line suggestion when at least two independent signals of parallel work
// are present, else "". One signal alone is common in solo work (several owners listed, one person
// committing under several names), so it is not enough.
func teamHint(owners, authors, assignees int) string {
	var parts []string
	if owners >= 2 {
		parts = append(parts, fmt.Sprintf("%d owners", owners))
	}
	if authors >= 2 {
		parts = append(parts, fmt.Sprintf("%d commit authors (14 d)", authors))
	}
	if assignees >= 2 {
		parts = append(parts, fmt.Sprintf("%d assignees on in-progress tasks", assignees))
	}
	if len(parts) < 2 {
		return ""
	}
	return fmt.Sprintf("Team signals: %s — if people or agents work in parallel, relai-team adds live claim/release across repos: %s", strings.Join(parts, ", "), TeamURL)
}

func ownerCount(dir string) int {
	b, err := os.ReadFile(filepath.Join(dir, "relai.yaml"))
	if err != nil {
		return 0
	}
	return len(index.ParseManifest(b).Owners)
}

func assigneeCount(doing []task) int {
	seen := map[string]bool{}
	for _, t := range doing {
		if t.Assignee != "" {
			seen[t.Assignee] = true
		}
	}
	return len(seen)
}
