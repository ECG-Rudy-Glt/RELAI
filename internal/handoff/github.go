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

// ghIssue is one open issue as `gh issue list --json` prints it. GitLab issues read with glab are
// converted to the same shape, so both backends share the mapping, the rendering and the budget.
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

// backendName and backendTool name the issue tracker and the command that reads it.
func backendName(b string) string {
	if b == "gitlab" {
		return "GitLab"
	}
	return "GitHub"
}

func backendTool(b string) string {
	if b == "gitlab" {
		return "glab"
	}
	return "gh"
}

// ghLimit is how many open issues are asked for. When that many come back, the total is "N+".
const ghLimit = 50

// issueTimeout bounds the one network call the handoff can make: a session must not wait on it.
var issueTimeout = 8 * time.Second

// ghHint and glHint are shown, instead of running anything, for a repository on that host that has
// not asked for the adapter.
const (
	ghHint = "to list the open GitHub issues here, add `tasks: github` to `relai.yaml`"
	glHint = "to list the open GitLab issues here, add `tasks: gitlab` to `relai.yaml`"
)

// issueList is what an issue adapter reports: the issues, or why they were not read, or how to turn
// the adapter on. used is false when nothing was run. problem is one of a short fixed set, never raw
// tool output.
type issueList struct {
	backend       string // "github" or "gitlab"
	issues        []ghIssue
	problem, hint string
	used          bool
}

// remoteIssues is the task adapter for repositories with no Beads: the open issues of the
// repository, read with the gh CLI (`tasks: github`) or the glab CLI (`tasks: gitlab`). It is the
// only network call a handoff can make, so it is made only when the repository asks for it in
// relai.yaml, and never when RELAI_NO_GH is set. The tool, not relai, knows the host and the token.
func remoteIssues(dir string) issueList {
	if os.Getenv("RELAI_NO_GH") != "" {
		return issueList{}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "relai.yaml"))
	switch index.ParseManifest(b).Tasks {
	case "github":
		out, problem := runIssueCLI(dir, "gh", "issue", "list", "--state", "open", "--limit", "50", "--json", "number,title,assignees,labels")
		l := issueList{backend: "github", problem: problem, used: true}
		if problem == "" {
			l.issues, l.problem = parseGitHub(out)
		}
		return l
	case "gitlab":
		// Open issues are glab's default; --output json is its JSON switch.
		out, problem := runIssueCLI(dir, "glab", "issue", "list", "--output", "json", "--per-page", "50")
		l := issueList{backend: "gitlab", problem: problem, used: true}
		if problem == "" {
			l.issues, l.problem = parseGitLab(out)
		}
		return l
	}
	remote := must(run(dir, "git", "remote", "get-url", "origin"))
	switch {
	case strings.Contains(remote, "github.com"):
		return issueList{hint: ghHint}
	case strings.Contains(strings.ToLower(remote), "gitlab"):
		return issueList{hint: glHint}
	}
	return issueList{}
}

// runIssueCLI runs one issue-listing command, once, bounded in time. The reason it gives when it
// cannot is one of a short fixed set, never the tool's own output.
func runIssueCLI(dir, tool string, args ...string) (out []byte, problem string) {
	if !onPath(tool) {
		return nil, "the " + tool + " command is not installed"
	}
	ctx, cancel := context.WithTimeout(context.Background(), issueTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, tool, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return nil, tool + " could not list the issues (offline, not signed in, or issues are disabled)"
	}
	return out, ""
}

func parseGitHub(out []byte) ([]ghIssue, string) {
	var issues []ghIssue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, "gh returned output that could not be read"
	}
	return issues, ""
}

// glIssue is one issue as `glab issue list --output json` prints it: the GitLab REST issue object.
// Every field is optional; iid (the number shown in the project), not id, is the issue's number.
type glIssue struct {
	IID       int      `json:"iid"`
	Title     string   `json:"title"`
	State     string   `json:"state"`
	Labels    []string `json:"labels"`
	Assignees []struct {
		Username string `json:"username"`
	} `json:"assignees"`
}

func parseGitLab(out []byte) ([]ghIssue, string) {
	var raw []glIssue
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, "glab returned output that could not be read"
	}
	issues := []ghIssue{}
	for _, g := range raw {
		if g.IID <= 0 || g.State == "closed" {
			continue
		}
		i := ghIssue{Number: g.IID, Title: g.Title}
		for _, a := range g.Assignees {
			if a.Username != "" {
				i.Assignees = append(i.Assignees, struct {
					Login string `json:"login"`
				}{a.Username})
			}
		}
		for _, l := range g.Labels {
			i.Labels = append(i.Labels, struct {
				Name string `json:"name"`
			}{l})
		}
		issues = append(issues, i)
	}
	return issues, ""
}
