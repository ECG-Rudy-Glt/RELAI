package handoff

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

func run(dir, name string, args ...string) (string, error) {
	if name == "git" {
		// Read-only, as the spec promises: --no-optional-locks stops `git status` from refreshing
		// .git/index; quotepath=off prints non-ASCII file names as they are, not as "caf\303\251".
		args = append([]string{"--no-optional-locks", "-c", "core.quotepath=off"}, args...)
	}
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if name == "git" {
		cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	}
	done := make(chan struct{})
	var out []byte
	var err error
	go func() { out, err = cmd.Output(); close(done) }()
	select {
	case <-done:
		return string(out), err
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		return "", fmt.Errorf("%s timed out", name)
	}
}

// cleanControl drops terminal control characters (ESC, BEL, NUL...) from text that comes from
// outside — commit subjects, file names, notes, task titles — so it cannot move the cursor, recolour
// the terminal or smuggle escape sequences to whoever reads or pastes the handoff. Newline-free
// text only: callers pass one line, a tab is kept.
func cleanControl(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isControl(r) {
			return -1
		}
		return r
	}, s)
}

func isControl(r rune) bool {
	return (r < 0x20 && r != '\t' && r != '\n') || r == 0x7f || (r >= 0x80 && r < 0xa0)
}

func must(s string, err error) string {
	if err != nil {
		return ""
	}
	return s
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}

// dedupe removes repeats from s, keeping first-seen order (unlike sorted-input dedupe elsewhere
// in this codebase, git log --name-only output is not sorted).
func dedupe(s []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, v := range s {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func head[T any](s []T, n int) []T {
	if n < len(s) {
		return s[:n]
	}
	return s
}

func short(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func onPath(bin string) bool { _, err := exec.LookPath(bin); return err == nil }
