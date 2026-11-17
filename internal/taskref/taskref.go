// Package taskref finds task ids in commit messages and tells whether a commit that names a task
// also left it open in its own copy of the task list.
package taskref

import (
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// idEnd ends a task id: the end of the text, a character that cannot continue an id, or a full stop
// that ends a sentence. A dot followed by a letter or digit continues the id, so "a-1" is not named
// by "a-1.2" (its child) and "a-1." at the end of a sentence still is.
const idEnd = `(?:$|[^A-Za-z0-9_.-]|\.(?:$|[^A-Za-z0-9_]))`

// idStart starts a task id: the start of the text or a character that cannot be part of an id. "#" is
// allowed in front ("#a-1"), as some people write.
const idStart = `(?:^|[^A-Za-z0-9_.-])#?`

// End is idEnd for callers that build their own pattern around an id.
const End = idEnd

// Mentions reports whether text names the task id as a whole id, not as the prefix of a longer one.
func Mentions(text, id string) bool {
	return regexp.MustCompile(idStart + regexp.QuoteMeta(id) + idEnd).MatchString(text)
}

// LeftOpen reports whether the commit sha changed the task id's own line in .beads/issues.jsonl
// (added it or edited it) and left it unfinished there. Such a commit created or updated the task, it
// says so itself: naming the task in its message is no sign of work done. A commit that only carries a
// fresh export (a pre-commit hook re-exports the list) without touching that task's line does not
// count, so "feat: CSV export (ex-1)" that forgot to close ex-1 is still reported. Errors answer false.
func LeftOpen(dir, sha, id string) bool {
	now, err := git(dir, "show", sha+":.beads/issues.jsonl")
	if err != nil {
		return false
	}
	line, status := taskLine(now, id)
	if line == "" || status == "closed" || status == "tombstone" {
		return false
	}
	before, err := git(dir, "show", sha+"^:.beads/issues.jsonl")
	if err != nil {
		return true // no parent, or the file is new in this commit: the task was added here
	}
	old, _ := taskLine(before, id)
	return old != line
}

// taskLine returns the line of file that holds the task id, and its status.
func taskLine(file, id string) (string, string) {
	for _, l := range strings.Split(file, "\n") {
		var t struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if json.Unmarshal([]byte(l), &t) == nil && t.ID == id {
			return strings.TrimSpace(l), t.Status
		}
	}
	return "", ""
}

func git(dir string, args ...string) (string, error) {
	// Read-only: no optional locks, so a handoff never refreshes .git/index.
	cmd := exec.Command("git", append([]string{"--no-optional-locks"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	return string(out), err
}
