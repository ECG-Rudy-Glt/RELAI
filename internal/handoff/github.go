package handoff

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

// ghIssue is one open GitHub issue as `gh issue list --json` prints it.
type ghIssue struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Assignees []struct {
		Login string `json:"login"`
	} `json:"assignees"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (i ghIssue) assignees() []string {
	var out []string
	for _, a := range i.Assignees {
		out = append(out, a.Login)
	}
	return out
}

func (i ghIssue) labels() []string {
	var out []string
	for _, l := range i.Labels {
		out = append(out, l.Name)
	}
	return out
}

// ghLimit is how many open issues are asked for. When that many come back, the total is "N+".
const ghLimit = 50

// ghTimeout bounds the one network call the handoff can make: a session must not wait on it.
const ghTimeout = 8 * time.Second

// ghHint is shown, instead of running anything, for a GitHub repository that has not asked for it.
const ghHint = "to list the open GitHub issues here, add `tasks: github` to `relai.yaml`"

// githubIssues is the task adapter for repositories with no Beads: the open issues of the
// repository, read with the gh CLI. It is the only network call a handoff can make, so it is made
// only when the repository asks for it with `tasks: github` in relai.yaml, and never when
// RELAI_NO_GH is set. used is false when nothing was run; hint then says how to turn it on, for a
// repository whose origin is on GitHub. problem is one of a short fixed set, never raw tool output.
func githubIssues(dir string) (issues []ghIssue, problem, hint string, used bool) {
	if os.Getenv("RELAI_NO_GH") != "" {
		return nil, "", "", false
	}
	b, _ := os.ReadFile(filepath.Join(dir, "relai.yaml"))
	if index.ParseManifest(b).Tasks != "github" {
		if strings.Contains(must(run(dir, "git", "remote", "get-url", "origin")), "github.com") {
			hint = ghHint
		}
		return nil, "", hint, false
	}
	if !onPath("gh") {
		return nil, "the gh command is not installed", "", true
	}
	ctx, cancel := context.WithTimeout(context.Background(), ghTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "issue", "list", "--state", "open", "--limit", "50", "--json", "number,title,assignees,labels")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, "gh could not list the issues (offline, not signed in, or issues are disabled)", "", true
	}
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, "gh returned output that could not be read", "", true
	}
	return issues, "", "", true
}
