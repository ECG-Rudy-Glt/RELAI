// Package handoff builds the compact, deterministic "where are we" summary of a repository.
//
// It only reads local, structured sources — git, Beads (`bd --json`), OpenSpec
// (`openspec list --json`) and the hand-curated `.relai/notes.md` — and never calls an LLM.
// The token budget is enforced by showing fewer items, never by rewording.
package handoff

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// FormatVersion is the version of spec/handoff.md this package implements.
const FormatVersion = "0.5"

// Options controls what goes into the handoff.
type Options struct {
	MaxTokens  int // hard budget for the whole document
	MaxCommits int // starting point, reduced if over budget
	MaxTasks   int // starting point, reduced if over budget
}

// DefaultOptions matches the repo contract: a handoff fits in 2,000 tokens.
func DefaultOptions() Options { return Options{MaxTokens: 2000, MaxCommits: 8, MaxTasks: 8} }

// EstimateTokens is a rough, model-agnostic estimate (~4 characters per token).
func EstimateTokens(s string) int { return (utf8.RuneCountInString(s) + 3) / 4 }

// Build returns the handoff Markdown for the repository at dir.
func Build(dir string, o Options) (string, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	top, err := run(dir, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", fmt.Errorf("%s is not a git repository", dir)
	}
	// Run from a subdirectory, the handoff is still the repository's: AGENTS.md, .beads and
	// .relai live at the root, and looking for them in the subdirectory called every such repo
	// "degraded".
	if root := strings.TrimSpace(top); root != "" {
		dir = root
	}
	s := collect(dir)
	name := filepath.Base(dir)
	// Cut order from spec/handoff.md §5: advice, task lists, commits, harness lines, degraded-mode
	// lists (longer first), then notes. Each stage finds, by bisection, how many items still fit,
	// instead of dropping one item and re-rendering the whole document each time (which was
	// quadratic and took minutes on a repo with many touched files or long notes).
	commits, tasks, harness := o.MaxCommits, o.MaxTasks, len(s.harness)
	recent, markers := len(s.recentFiles), len(s.markers)
	notes := -1
	fits := func() (string, bool) {
		doc := render(name, s, commits, tasks, harness, recent, markers, notes)
		return doc, EstimateTokens(doc) <= o.MaxTokens
	}
	if doc, ok := fits(); ok {
		return doc, nil
	}
	if s.teamHint != "" { // advice, not state: always the first thing to go
		s.teamHint = ""
		if doc, ok := fits(); ok {
			return doc, nil
		}
	}
	// largest value in [lo, hi] for which set(v) makes the document fit; lo when none does.
	largest := func(lo, hi int, set func(int)) {
		hi0 := hi
		for lo < hi {
			mid := (lo + hi + 1) / 2
			set(mid)
			if _, ok := fits(); ok {
				lo = mid
			} else {
				hi = mid - 1
			}
		}
		// The size is not strictly monotonic: the "… (N more)" line gets shorter when N gains a
		// digit. Probe a few values above the bisection result so the answer is the same as
		// the largest value found by trying them all.
		best := lo
		for v := lo + 1; v <= hi0 && v <= lo+8; v++ {
			set(v)
			if _, ok := fits(); ok {
				best = v
			}
		}
		set(best)
	}
	if tasks > 3 {
		largest(3, tasks, func(v int) { tasks = v })
		if doc, ok := fits(); ok {
			return doc, nil
		}
	}
	if commits > 3 {
		largest(3, commits, func(v int) { commits = v })
		if doc, ok := fits(); ok {
			return doc, nil
		}
	}
	if harness > 0 {
		largest(0, harness, func(v int) { harness = v })
		if doc, ok := fits(); ok {
			return doc, nil
		}
	}
	// Degraded-mode lists shrink together, the longer one first (ties cut the files list).
	// Precompute the sequence of (recent, markers) states; no rendering needed for that.
	type pair struct{ r, m int }
	states := []pair{{recent, markers}}
	for r, m := recent, markers; r > 0 || m > 0; {
		if r > 0 && r >= m {
			r--
		} else {
			m--
		}
		states = append(states, pair{r, m})
	}
	lo, hi := 0, len(states)-1 // smallest index (fewest cuts) that fits
	for lo < hi {
		mid := (lo + hi) / 2
		recent, markers = states[mid].r, states[mid].m
		if _, ok := fits(); ok {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	for i := lo - 1; i >= 0 && i >= lo-8; i-- { // same non-monotonic probe as above
		recent, markers = states[i].r, states[i].m
		if _, ok := fits(); ok {
			lo = i
		}
	}
	recent, markers = states[lo].r, states[lo].m
	if doc, ok := fits(); ok {
		return doc, nil
	}
	// Last resort: cut the curated notes, which are the only free-form part.
	if len(s.notes) > 0 {
		largest(0, len(s.notes), func(v int) { notes = v })
	} else {
		notes = 0
	}
	doc, _ := fits()
	return doc, nil
}
