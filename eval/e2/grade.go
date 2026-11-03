package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// Expected is a fixture's expected.json: what counts as the right next step, and the known wrong
// answers ("traps") with the reason they are wrong.
type Expected struct {
	Summary        string            `json:"summary"`
	AcceptIDs      []string          `json:"accept_ids"`
	AcceptPatterns []string          `json:"accept_patterns"`
	Traps          map[string]string `json:"traps"` // task ID or regexp → reason
}

func loadExpected(path string) (Expected, error) {
	var e Expected
	b, err := os.ReadFile(path)
	if err != nil {
		return e, err
	}
	return e, json.Unmarshal(b, &e)
}

// NextLine returns the text after the last "NEXT:" in the agent's answer.
func NextLine(answer string) string {
	i := strings.LastIndex(answer, "NEXT:")
	if i < 0 {
		return ""
	}
	line, _, _ := strings.Cut(answer[i+len("NEXT:"):], "\n")
	return strings.TrimSpace(strings.NewReplacer("`", "", "**", "").Replace(line))
}

// Grade decides whether a NEXT line is right, in this order: an accepted task ID, a trap task ID,
// an accepted pattern, a trap pattern. Task IDs win over wording, so "lc-3 — worker pool" is wrong;
// an accepted pattern wins over a trap pattern, so "fix the contact form, the blog is dropped" is
// right.
func (e Expected) Grade(next string) (correct bool, trap string) {
	if next == "" {
		return false, "no NEXT line"
	}
	if len(e.AcceptIDs) == 0 && len(e.AcceptPatterns) == 0 {
		return false, ungraded
	}
	for _, id := range e.AcceptIDs {
		if matches(id, next) {
			return true, ""
		}
	}
	for k, reason := range e.Traps {
		if !isPattern(k) && matches(k, next) {
			return false, reason
		}
	}
	for _, p := range e.AcceptPatterns {
		if matches(p, next) {
			return true, ""
		}
	}
	for k, reason := range e.Traps {
		if isPattern(k) && matches(k, next) {
			return false, reason
		}
	}
	return false, "other: " + short(next, 60)
}

// ungraded marks answers on real repos, which have no known right answer.
const ungraded = "ungraded"

func isPattern(key string) bool { return strings.HasPrefix(key, "(?") }

// matches treats keys starting with "(?" as regexps and anything else as a whole-word task ID.
func matches(key, s string) bool {
	if isPattern(key) {
		return regexp.MustCompile(key).MatchString(s)
	}
	return regexp.MustCompile(`(?i)(^|[^\w-])` + regexp.QuoteMeta(key) + `($|[^\w-])`).MatchString(s)
}
