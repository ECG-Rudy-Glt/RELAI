// Package tidy does the housekeeping that otherwise depends on someone remembering it after work is
// merged: closing the tasks a merged commit says it closes, refreshing the committed task export, and
// deleting local branches that are fully merged. It acts only on explicit evidence and is a dry run
// unless told to apply.
//
// Evidence for closing a task is a closing keyword in a commit message on the default branch followed by
// the task's id ("Closes relai-7nk", "fixes #relai-7nk"). A task merely named in a commit is reported,
// never closed: it may be half done.
package tidy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
)

// Closure is a task a commit on the default branch says it closes.
type Closure struct {
	Task    string `json:"task"`
	Commit  string `json:"commit"`
	Subject string `json:"subject"`
}

// Plan is what tidy would do.
type Plan struct {
	Default   string    `json:"default_branch"`
	OnDefault bool      `json:"on_default_branch"`
	Close     []Closure `json:"close"`
	Named     []Closure `json:"named_not_closed"` // open tasks named in a commit without a closing keyword
	Branches  []string  `json:"merged_branches"`
}

// Result is what Apply did.
type Result struct {
	Closed          []string
	DeletedBranches []string
	ExportRewritten bool
	Notes           []string
}

var protected = map[string]bool{"main": true, "master": true, "develop": true, "dev": true, "trunk": true}

// Build looks at the last `limit` commits of dir's current branch (default 300).
func Build(dir string, limit int) (Plan, error) {
	var p Plan
	if limit <= 0 {
		limit = 300
	}
	if _, err := git(dir, "rev-parse", "--git-dir"); err != nil {
		return p, fmt.Errorf("%s is not a git repository", dir)
	}
	p.Default = defaultBranch(dir)
	cur, _ := git(dir, "branch", "--show-current")
	p.OnDefault = strings.TrimSpace(cur) == p.Default

	open := map[string]bool{}
	for _, t := range handoff.SnapshotBrief(dir) {
		open[t.ID] = true
	}
	if len(open) > 0 {
		out, err := git(dir, "log", fmt.Sprintf("-%d", limit), "--format=%h%x1f%s%x1f%B%x1e")
		if err == nil {
			closed, named := map[string]bool{}, map[string]bool{}
			for _, rec := range strings.Split(out, "\x1e") {
				f := strings.SplitN(strings.TrimSpace(rec), "\x1f", 3)
				if len(f) < 3 {
					continue
				}
				sha, subject, body := f[0], f[1], f[2]
				for id := range open {
					if closed[id] {
						continue
					}
					if closes(body, id) {
						closed[id] = true
						p.Close = append(p.Close, Closure{Task: id, Commit: sha, Subject: subject})
					} else if !named[id] && mentions(body, id) {
						named[id] = true
						p.Named = append(p.Named, Closure{Task: id, Commit: sha, Subject: subject})
					}
				}
			}
			// a task that is closed by a later commit is not also "named"
			var n []Closure
			for _, c := range p.Named {
				if !closed[c.Task] {
					n = append(n, c)
				}
			}
			p.Named = n
		}
	}
	if !p.OnDefault {
		// Work on a feature branch is not merged yet: closing its tasks now would be wrong.
		p.Close = nil
	}
	if out, err := git(dir, "branch", "--merged", p.Default, "--format=%(refname:short)"); err == nil {
		for _, b := range strings.Split(out, "\n") {
			b = strings.TrimSpace(b)
			if b != "" && b != p.Default && b != strings.TrimSpace(cur) && !protected[b] {
				p.Branches = append(p.Branches, b)
			}
		}
	}
	return p, nil
}

func closes(msg, id string) bool {
	re := regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\b[:\s]+(?:[^\n]*?[\s,]|)#?` + regexp.QuoteMeta(id) + `\b`)
	return re.MatchString(msg)
}

func mentions(msg, id string) bool {
	return regexp.MustCompile(`(^|[^A-Za-z0-9_.-])#?` + regexp.QuoteMeta(id) + `($|[^A-Za-z0-9_-])`).MatchString(msg)
}

func defaultBranch(dir string) string {
	if out, err := git(dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		return strings.TrimPrefix(strings.TrimSpace(out), "origin/")
	}
	for _, b := range []string{"main", "master"} {
		if _, err := git(dir, "rev-parse", "--verify", "-q", "refs/heads/"+b); err == nil {
			return b
		}
	}
	cur, _ := git(dir, "branch", "--show-current")
	return strings.TrimSpace(cur)
}

// Apply does what the plan says: closes the tasks with bd (a reason names the commit), rewrites
// .beads/issues.jsonl from bd without the owner field, and deletes the merged branches with `git
// branch -d`, which refuses anything not fully merged. It does not commit.
func (p Plan) Apply(dir string) Result {
	var r Result
	if len(p.Close) > 0 {
		if _, err := exec.LookPath("bd"); err != nil {
			r.Notes = append(r.Notes, "bd is not installed: tasks not closed")
		} else {
			for _, c := range p.Close {
				if out, err := run(dir, 60*time.Second, "bd", "close", c.Task, "--reason", "closed by commit "+c.Commit); err != nil {
					r.Notes = append(r.Notes, fmt.Sprintf("could not close %s: %s", c.Task, firstLine(out)))
					continue
				}
				r.Closed = append(r.Closed, c.Task)
			}
			if len(r.Closed) > 0 {
				if err := exportTasks(dir); err != nil {
					r.Notes = append(r.Notes, "tasks closed but the export was not refreshed: "+err.Error())
				} else {
					r.ExportRewritten = true
				}
			}
		}
	}
	for _, b := range p.Branches {
		if _, err := git(dir, "branch", "-d", b); err == nil {
			r.DeletedBranches = append(r.DeletedBranches, b)
		} else {
			r.Notes = append(r.Notes, "kept branch "+b+" (git says it is not fully merged)")
		}
	}
	return r
}

var ownerRE = regexp.MustCompile(`"owner":"[^"]*",`)

// exportTasks rewrites .beads/issues.jsonl from bd, dropping the owner field (the creator's e-mail).
func exportTasks(dir string) error {
	tmp, err := os.CreateTemp("", "relai-export-*")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	if out, err := run(dir, 60*time.Second, "bd", "export", "-o", tmp.Name()); err != nil {
		return fmt.Errorf("bd export: %s", firstLine(out))
	}
	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		return err
	}
	b = ownerRE.ReplaceAll(b, nil)
	dst := filepath.Join(dir, ".beads", "issues.jsonl")
	if err := os.WriteFile(dst+".tmp", b, 0o644); err != nil {
		return err
	}
	return os.Rename(dst+".tmp", dst)
}

// Text renders the plan for a person.
func (p Plan) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "tidy (default branch: %s%s)\n", p.Default, map[bool]string{true: "", false: "; you are on another branch, so no task is closed from here"}[p.OnDefault])
	if len(p.Close) == 0 && len(p.Named) == 0 && len(p.Branches) == 0 {
		b.WriteString("nothing to do.\n")
		return b.String()
	}
	for _, c := range p.Close {
		fmt.Fprintf(&b, "- close %s (commit %s: %s)\n", c.Task, c.Commit, c.Subject)
	}
	for _, c := range p.Named {
		fmt.Fprintf(&b, "- %s is named in commit %s without \"closes\": left open, check it by hand (%s)\n", c.Task, c.Commit, c.Subject)
	}
	for _, br := range p.Branches {
		fmt.Fprintf(&b, "- delete merged local branch %s\n", br)
	}
	return b.String()
}

// git returns standard output only: a warning on stderr must not end up in parsed output.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"--no-optional-locks"}, args...)...)
	cmd.Dir = dir
	done := make(chan struct{})
	var out []byte
	var err error
	go func() { out, err = cmd.Output(); close(done) }()
	select {
	case <-done:
		return string(out), err
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("git timed out")
	}
}

func run(dir string, limit time.Duration, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	done := make(chan struct{})
	var out []byte
	var err error
	go func() { out, err = cmd.CombinedOutput(); close(done) }()
	select {
	case <-done:
		return string(out), err
	case <-time.After(limit):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("%s timed out", name)
	}
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return s
}
