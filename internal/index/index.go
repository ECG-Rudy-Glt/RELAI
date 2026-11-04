// Package index builds and reads a flat, git-only cross-repo index of relai.yaml manifests, so
// relai find can tell a user whether a capability already exists in another one of their
// projects before they build a duplicate.
//
// Index building is read-only and pull-based: it only reads each listed repo's relai.yaml, never
// writes to it, and never requires push access or a hosted service.
package index

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Capability is one provides:/consumes: entry of a relai.yaml.
type Capability struct {
	Capability string `json:"capability"`
}

// Manifest is the subset of relai.yaml the index cares about.
type Manifest struct {
	APIVersion string       `json:"apiVersion,omitempty"`
	ID         string       `json:"id,omitempty"`
	Summary    string       `json:"summary,omitempty"`
	Owners     []string     `json:"owners,omitempty"`
	Provides   []Capability `json:"provides,omitempty"`
	Consumes   []Capability `json:"consumes,omitempty"`
}

// Entry is one indexed repo: where it came from and its parsed manifest.
type Entry struct {
	Source    string   `json:"source"`
	RelaiYAML Manifest `json:"relai_yaml"`
}

// Skip records a listed source that could not be indexed, and why.
type Skip struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// Find returns every entry whose relai.yaml lists capability under provides:.
func Find(entries []Entry, capability string) []Entry {
	var out []Entry
	for _, e := range entries {
		for _, p := range e.RelaiYAML.Provides {
			if strings.EqualFold(p.Capability, capability) {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// ReadList reads a repos.txt-style file: one path or clone URL per line, blank lines and lines
// starting with # ignored.
func ReadList(path string) ([]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		out = append(out, l)
	}
	return out, nil
}

// Build reads each source's relai.yaml (cloning clone-URL sources to a temp dir first) and
// returns the entries it could read plus the sources it had to skip, with a reason for each.
// It never fails outright: a bad source is reported as a skip, not an error.
func Build(sources []string) (entries []Entry, skipped []Skip) {
	for _, src := range sources {
		m, err := readOne(src)
		if err != nil {
			skipped = append(skipped, Skip{Source: src, Reason: err.Error()})
			continue
		}
		entries = append(entries, Entry{Source: src, RelaiYAML: m})
	}
	return entries, skipped
}

func readOne(src string) (Manifest, error) {
	path, cleanup, err := Fetch(src)
	if err != nil {
		return Manifest{}, err
	}
	defer cleanup()
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return Manifest{}, fmt.Errorf("not found")
	}
	if _, err := exec.Command("git", "-C", path, "rev-parse", "--git-dir").Output(); err != nil {
		return Manifest{}, fmt.Errorf("not a git repo")
	}
	b, err := os.ReadFile(filepath.Join(path, "relai.yaml"))
	if err != nil {
		return Manifest{}, fmt.Errorf("no relai.yaml")
	}
	return parseManifest(b), nil
}

var cloneURLPattern = regexp.MustCompile(`^(https?|git|ssh|file)://|^git@`)

func isCloneURL(s string) bool { return cloneURLPattern.MatchString(s) }

// Fetch resolves src (a local path or clone URL) to a local directory: a clone URL is
// shallow-cloned to a new temp directory, read-only; a local path is returned as-is. The caller
// MUST call cleanup once done with the path; it is a no-op for a local path. Shared by relai
// index build and relai new (internal/scaffold), so both source a repo or template the same way.
func Fetch(src string) (path string, cleanup func(), err error) {
	if !isCloneURL(src) {
		return src, func() {}, nil
	}
	tmp, err := cloneShallow(src)
	if err != nil {
		return "", func() {}, err
	}
	return tmp, func() { os.RemoveAll(tmp) }, nil
}

// cloneShallow clones a URL (including file://) to a new temp dir and returns its path; the
// caller removes it.
func cloneShallow(url string) (string, error) {
	tmp, err := os.MkdirTemp("", "relai-index-*")
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "clone", "--depth", "1", "--quiet", url, tmp)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()
	select {
	case err := <-done:
		if err != nil {
			os.RemoveAll(tmp)
			return "", fmt.Errorf("clone %s: %w%s", url, err, gitReason(stderr.String()))
		}
		return tmp, nil
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		os.RemoveAll(tmp)
		return "", fmt.Errorf("clone %s: timed out", url)
	}
}

// ParseManifest parses the relai.yaml subset described on parseManifest. It never fails.
func ParseManifest(b []byte) Manifest { return parseManifest(b) }

// parseManifest reads the small subset of YAML that relai.yaml uses: flat scalar keys, inline
// bracket lists, and one-level block lists of single-key maps (provides:/consumes:). It never
// fails; keys it cannot parse are left at their zero value, matching the project's "stdlib only,
// no new dependency for a format this constrained" stance.
func parseManifest(b []byte) Manifest {
	var m Manifest
	lines := strings.Split(string(b), "\n")
	for i := 0; i < len(lines); i++ {
		key, rest, ok := splitKV(lines[i])
		if !ok {
			continue
		}
		switch key {
		case "apiVersion":
			m.APIVersion = unquote(rest)
		case "id":
			m.ID = unquote(rest)
		case "summary":
			m.Summary = unquote(rest)
		case "owners":
			if rest != "" {
				m.Owners = inlineList(rest)
			}
		case "provides":
			var caps []Capability
			i, caps = blockList(lines, i)
			m.Provides = caps
		case "consumes":
			var caps []Capability
			i, caps = blockList(lines, i)
			m.Consumes = caps
		}
	}
	return m
}

// splitKV splits a top-level "key: value" line (no leading indentation), stripping a trailing
// unquoted comment. ok is false for blank lines, comments, or indented (nested) lines.
func splitKV(line string) (key, value string, ok bool) {
	if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' {
		return "", "", false
	}
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = stripComment(strings.TrimSpace(line[idx+1:]))
	return key, value, true
}

// blockList consumes the "  - capability: x" lines following a "provides:"/"consumes:" header at
// lines[start], returning the index of the last line consumed and the parsed capabilities.
func blockList(lines []string, start int) (int, []Capability) {
	var out []Capability
	i := start
	for i+1 < len(lines) {
		l := lines[i+1]
		trimmed := strings.TrimSpace(l)
		if l == "" || (len(l) > 0 && (l[0] != ' ' && l[0] != '\t')) {
			break
		}
		if !strings.HasPrefix(trimmed, "- ") {
			break
		}
		if _, v, ok := splitKV(strings.TrimPrefix(trimmed, "- ")); ok {
			out = append(out, Capability{Capability: unquote(v)})
		}
		i++
	}
	return i, out
}

func inlineList(s string) []string {
	s = stripComment(s)
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		out = append(out, unquote(strings.TrimSpace(part)))
	}
	return out
}

func stripComment(s string) string {
	inQuote := byte(0)
	for i := 0; i < len(s); i++ {
		switch {
		case inQuote != 0:
			if s[i] == inQuote {
				inQuote = 0
			}
		case s[i] == '\'' || s[i] == '"':
			inQuote = s[i]
		case s[i] == '#':
			return strings.TrimSpace(s[:i])
		}
	}
	return strings.TrimSpace(s)
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// gitReason keeps the last meaningful line of git's stderr ("fatal: repository ... not found") as
// ": <reason>", with control characters removed, so a failed clone says why.
func gitReason(stderr string) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	last = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, last)
	if last == "" {
		return ""
	}
	if len(last) > 200 {
		last = last[:200] + "…"
	}
	return ": " + last
}
