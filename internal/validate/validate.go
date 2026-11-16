// Package validate checks a repository against the repo contract: relai.yaml as specified in
// spec/relai-yaml.md, the size of AGENTS.md, and the presence of .relai/notes.md.
package validate

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

// Levels of a finding. An error makes `relai validate` fail; a warning does only with --strict.
const (
	Error   = "error"
	Warning = "warning"
)

// MaxAgentsLines is the budget of AGENTS.md (docs/DESIGN.md §4: read at every session, so kept short).
const MaxAgentsLines = 100

// MaxSummaryRunes is the length above which a summary is no longer "one sentence".
const MaxSummaryRunes = 200

// APIVersion is the format version this build reads.
const APIVersion = "relai.dev/v0"

// Finding is one thing validate has to say about a file.
type Finding struct {
	Level   string `json:"level"`
	File    string `json:"file"`
	Message string `json:"message"`
}

var (
	idPattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
	capabilityPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
)

// Run checks the repository at dir and returns what it found, in a stable order.
func Run(dir string) []Finding {
	var out []Finding
	add := func(level, file, format string, a ...any) {
		out = append(out, Finding{Level: level, File: file, Message: fmt.Sprintf(format, a...)})
	}
	manifest(dir, add)

	const agents = "AGENTS.md"
	if b, err := os.ReadFile(filepath.Join(dir, agents)); err != nil {
		add(Warning, agents, "not found: agents have no entry point (relai init writes one)")
	} else if n := lineCount(b); n > MaxAgentsLines {
		add(Error, agents, "%d lines, the budget is %d: it is read at every session, move details to other files and link them", n, MaxAgentsLines)
	}
	const notes = ".relai/notes.md"
	if _, err := os.Stat(filepath.Join(dir, notes)); err != nil {
		add(Warning, notes, "not found: decisions and pitfalls have nowhere to live (relai init writes one)")
	}
	return out
}

func manifest(dir string, add func(level, file, format string, a ...any)) {
	const file = "relai.yaml"
	b, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		add(Error, file, "not found (relai init writes one)")
		return
	}
	m, warns := index.ParseManifestWarn(b)
	switch {
	case m.APIVersion == "":
		add(Warning, file, "apiVersion is missing: add `apiVersion: %s`", APIVersion)
	case m.APIVersion != APIVersion:
		add(Error, file, "apiVersion %q is not a version this relai reads (%s)", m.APIVersion, APIVersion)
	}
	switch {
	case m.ID == "":
		add(Error, file, "id is missing or empty")
	case !idPattern.MatchString(m.ID):
		add(Error, file, "id %q is not a name (letters, digits, . _ - /, starting with a letter or digit)", m.ID)
	}
	switch n := len([]rune(m.Summary)); {
	case strings.TrimSpace(m.Summary) == "":
		add(Error, file, "summary is missing or empty: one sentence that says what this repository is for")
	case n > MaxSummaryRunes:
		add(Warning, file, "summary is %d characters long: keep it to one sentence (%d at most)", n, MaxSummaryRunes)
	}
	// A value the reader did not understand is a capability that silently disappears from the index.
	for _, w := range warns {
		add(Error, file, "%s", w)
	}
	names := func(key string, caps []index.Capability) {
		seen := map[string]bool{}
		for _, c := range caps {
			switch {
			case !capabilityPattern.MatchString(c.Capability):
				add(Warning, file, "%s: %q is not a dotted lowercase name such as auth.oidc", key, c.Capability)
			case seen[c.Capability]:
				add(Warning, file, "%s: %q is listed twice", key, c.Capability)
			}
			seen[c.Capability] = true
		}
	}
	names("provides", m.Provides)
	names("consumes", m.Consumes)
	for _, l := range m.Links {
		if !idPattern.MatchString(l) {
			add(Warning, file, "links: %q is not the id of a repository", l)
		}
	}
	known := map[string]bool{}
	for _, k := range index.KnownKeys {
		known[k] = true
	}
	seen := map[string]bool{}
	for _, k := range index.TopLevelKeys(b) {
		switch {
		case !known[k] && !seen[k]:
			add(Warning, file, "unknown key %q: readers ignore it (a typo?)", k)
		case known[k] && seen[k] && k != "provides" && k != "consumes": // those two are reported by the reader
			add(Error, file, "the key %q appears twice", k)
		}
		seen[k] = true
	}
}

func lineCount(b []byte) int {
	if len(b) == 0 {
		return 0
	}
	n := bytes.Count(b, []byte("\n"))
	if b[len(b)-1] != '\n' {
		n++
	}
	return n
}

// Failed reports whether the findings make the check fail: any error, or any finding when strict.
func Failed(f []Finding, strict bool) bool {
	for _, x := range f {
		if strict || x.Level == Error {
			return true
		}
	}
	return false
}

// Text renders the findings for a person.
func Text(f []Finding) string {
	if len(f) == 0 {
		return "ok: relai.yaml, AGENTS.md and .relai/notes.md follow the contract.\n"
	}
	var b strings.Builder
	errs := 0
	for _, x := range f {
		if x.Level == Error {
			errs++
		}
		fmt.Fprintf(&b, "%s: %s: %s\n", x.File, x.Level, x.Message)
	}
	fmt.Fprintf(&b, "%d error(s), %d warning(s).\n", errs, len(f)-errs)
	return b.String()
}
