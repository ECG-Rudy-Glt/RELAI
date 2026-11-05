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
	"github.com/ECG-Rudy-Glt/RELAI/internal/scan"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
	// Links are the ids of the repositories this one works with, declared by a person after checking
	// the mentions `relai discover --links` found. Capabilities say what; links say who.
	Links []string `json:"links,omitempty"`
}

// Entry is one indexed repo: where it came from and its parsed manifest.
type Entry struct {
	Source    string   `json:"source"`
	RelaiYAML Manifest `json:"relai_yaml"`
	// Warnings lists what in the repo's relai.yaml could not be understood (never silently
	// dropped); it is shown by `relai index build` and kept in the index.
	Warnings []string `json:"warnings,omitempty"`
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

// Relation says how an indexed repo relates to the repo `me` (the one the person is working in), from the
// links: each side declared: "linked" (me lists it), "links to you" (it lists me), "both" or "".
func Relation(e Entry, me Manifest) string {
	mine, theirs := false, false
	for _, l := range me.Links {
		if me.ID != "" && l == e.RelaiYAML.ID {
			mine = true
		}
	}
	for _, l := range e.RelaiYAML.Links {
		if me.ID != "" && l == me.ID {
			theirs = true
		}
	}
	switch {
	case mine && theirs:
		return "linked both ways"
	case mine:
		return "linked from here"
	case theirs:
		return "links to this repo"
	}
	return ""
}

// Order puts the repos related to `me` first, keeping the given order otherwise (it is stable and never
// removes anything).
func Order(found []Entry, me Manifest) []Entry {
	out := append([]Entry(nil), found...)
	sort.SliceStable(out, func(i, j int) bool {
		return Relation(out[i], me) != "" && Relation(out[j], me) == ""
	})
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
	first := map[string]string{} // identity of a repository -> the source that named it first
	for _, src := range sources {
		// The same repository named twice (a list file and a scanned folder, a symbolic link, a worktree) is
		// indexed once, and the second name is reported.
		id := scan.Identity(src)
		if prev, dup := first[id]; dup {
			skipped = append(skipped, Skip{Source: src, Reason: "same repository as " + prev})
			continue
		}
		first[id] = src
		m, warn, err := readOne(src)
		if err != nil {
			skipped = append(skipped, Skip{Source: src, Reason: err.Error()})
			continue
		}
		entries = append(entries, Entry{Source: src, RelaiYAML: m, Warnings: warn})
	}
	return entries, skipped
}

func readOne(src string) (Manifest, []string, error) {
	path, cleanup, err := Fetch(src)
	if err != nil {
		return Manifest{}, nil, err
	}
	defer cleanup()
	st, err := os.Stat(path)
	if err != nil || !st.IsDir() {
		return Manifest{}, nil, fmt.Errorf("not found")
	}
	if _, err := exec.Command("git", "-C", path, "rev-parse", "--git-dir").Output(); err != nil {
		return Manifest{}, nil, fmt.Errorf("not a git repo")
	}
	b, err := os.ReadFile(filepath.Join(path, "relai.yaml"))
	if err != nil {
		return Manifest{}, nil, fmt.Errorf("no relai.yaml")
	}
	m, warn := parseManifest(b)
	return m, warn, nil
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
func ParseManifest(b []byte) Manifest {
	m, _ := ParseManifestWarn(b)
	return m
}

// ParseManifestWarn is ParseManifest plus a list of things it could not understand, so a caller
// can say so instead of letting a capability vanish silently (which makes relai find answer
// "nothing provides X" and invites a duplicate).
func ParseManifestWarn(b []byte) (Manifest, []string) { return parseManifest(b) }

// parseManifest reads the small subset of YAML that relai.yaml uses: flat scalar keys, inline
// bracket lists, and provides:/consumes: lists in any of these shapes:
//
//	provides:                      provides: [a, b]
//	  - capability: a              provides: [{capability: a}, {capability: b}]
//	  - a                          provides:
//	  # comments and blank lines      - id: x
//	  - id: x                          capability: a   (capability: in any key position)
//	    capability: b
//
// It never fails; a key it cannot make sense of is reported in the warnings and left at its zero
// value. That matches the project's "stdlib only, no dependency for a format this constrained"
// stance, with the rule that nothing is dropped without a message.
func parseManifest(b []byte) (Manifest, []string) {
	var m Manifest
	var warn []string
	seenKeys := map[string]bool{}
	text := strings.TrimPrefix(string(b), "\ufeff")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		key, rest, ok := splitKV(lines[i])
		if !ok {
			continue
		}
		switch key {
		case "apiVersion":
			m.APIVersion = unquote(rest)
		case "id":
			m.ID = scalar(rest, "id", &warn)
		case "summary":
			m.Summary = unquote(rest)
		case "owners":
			if rest != "" {
				m.Owners = inlineList(rest)
			}
		case "links":
			var caps []Capability
			var w []string
			i, caps, w = capabilityList(lines, i, rest)
			for _, x := range w {
				warn = append(warn, "links: "+x)
			}
			for _, c := range caps {
				m.Links = append(m.Links, c.Capability)
			}
		case "provides", "consumes":
			var caps []Capability
			var w []string
			i, caps, w = capabilityList(lines, i, rest)
			for _, x := range w {
				warn = append(warn, key+": "+x)
			}
			if len(caps) == 0 && len(w) == 0 && rest != "" && rest != "[]" && rest != "~" && rest != "null" {
				warn = append(warn, key+": value "+rest+" was not understood")
			}
			// A key written twice is invalid YAML. Keep both lists (losing the first would make `relai find`
			// say nothing provides it) and say so.
			if seenKeys[key] {
				warn = append(warn, key+": the key appears twice; both lists are kept, merge them")
			}
			seenKeys[key] = true
			if key == "provides" {
				m.Provides = append(m.Provides, caps...)
			} else {
				m.Consumes = append(m.Consumes, caps...)
			}
		}
	}
	return m, warn
}

// scalar returns an unquoted scalar, refusing a value that looks like an unfinished flow
// collection (id: [unclosed), which would otherwise be used as if it were a name.
func scalar(rest, key string, warn *[]string) string {
	v := unquote(rest)
	if len(v) > 0 && (v[0] == '[' || v[0] == '{') && rest == v {
		*warn = append(*warn, key+": value "+rest+" is not a plain name")
		return ""
	}
	return v
}

// splitKV splits a top-level "key: value" line (no leading indentation), stripping a trailing
// unquoted comment. ok is false for blank lines, comments, or indented (nested) lines.
func splitKV(line string) (key, value string, ok bool) {
	if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' || line[0] == '-' {
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

// capabilityList reads the value of a provides:/consumes: key whose header is at lines[start]
// (rest is what follows the colon), returning the index of the last line consumed.
func capabilityList(lines []string, start int, rest string) (int, []Capability, []string) {
	var caps []Capability
	var warn []string
	if rest != "" {
		if !strings.HasPrefix(rest, "[") || !strings.HasSuffix(rest, "]") {
			return start, nil, nil // not a list; the caller reports it
		}
		inner := strings.TrimSpace(rest[1 : len(rest)-1])
		for _, part := range splitTop(inner) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
				found := false
				for _, kv := range splitTop(part[1 : len(part)-1]) {
					if k, v, ok := keyValue(kv); ok && k == "capability" {
						caps = append(caps, Capability{Capability: unquote(v)})
						found = true
					}
				}
				if !found {
					warn = append(warn, "entry "+part+" has no capability: key")
				}
				continue
			}
			caps = append(caps, Capability{Capability: unquote(part)})
		}
		return start, caps, warn
	}
	i := start
	inItem, got := false, false
	for i+1 < len(lines) {
		l := lines[i+1]
		t := strings.TrimSpace(stripComment(l))
		raw := strings.TrimSpace(l)
		switch {
		case raw == "" || strings.HasPrefix(raw, "#"):
			i++
			continue
		case !strings.HasPrefix(l, " ") && !strings.HasPrefix(l, "\t") && !strings.HasPrefix(l, "-"):
			return i, caps, warn // next top-level key
		case strings.HasPrefix(t, "- ") || t == "-":
			if inItem && !got {
				warn = append(warn, "a list entry has no capability: key")
			}
			inItem, got = true, false
			item := strings.TrimSpace(strings.TrimPrefix(t, "-"))
			if item == "" {
				break
			}
			if k, v, ok := keyValue(item); ok && !strings.HasPrefix(item, "\"") && !strings.HasPrefix(item, "'") {
				if k == "capability" {
					caps = append(caps, Capability{Capability: unquote(v)})
					got = true
				}
			} else {
				caps = append(caps, Capability{Capability: unquote(item)})
				got = true
			}
		default: // continuation line of the current entry
			if k, v, ok := keyValue(t); ok && k == "capability" && inItem && !got {
				caps = append(caps, Capability{Capability: unquote(v)})
				got = true
			}
		}
		i++
	}
	if inItem && !got {
		warn = append(warn, "a list entry has no capability: key")
	}
	return i, caps, warn
}

// keyValue splits "key: value" (any indentation) without treating a quoted string as a key.
func keyValue(s string) (key, value string, ok bool) {
	s = strings.TrimSpace(s)
	idx := strings.Index(s, ":")
	if idx <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(s[:idx]), stripComment(strings.TrimSpace(s[idx+1:])), true
}

// splitTop splits on commas that are not inside braces, brackets or quotes.
func splitTop(s string) []string {
	var out []string
	depth, start := 0, 0
	var q byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case q != 0:
			if c == q {
				q = 0
			}
		case c == '"' || c == '\'':
			q = c
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
		case c == ',' && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
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
