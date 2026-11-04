// Package handoff builds the compact, deterministic "where are we" summary of a repository.
//
// It only reads local, structured sources — git, Beads (`bd --json`), OpenSpec
// (`openspec list --json`) and the hand-curated `.relai/notes.md` — and never calls an LLM.
// The token budget is enforced by showing fewer items, never by rewording.
package handoff

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ECG-Rudy-Glt/RELAI/internal/discover"
	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
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
	if _, err := run(dir, "git", "rev-parse", "--git-dir"); err != nil {
		return "", fmt.Errorf("%s is not a git repository", dir)
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

type task struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Priority  int      `json:"priority"`
	Assignee  string   `json:"assignee"`
	BlockedBy []string `json:"blocked_by"`
}

type change struct {
	Name           string `json:"name"`
	CompletedTasks int    `json:"completedTasks"`
	TotalTasks     int    `json:"totalTasks"`
	Status         string `json:"status"`
	Next           string `json:"-"` // first unchecked task in tasks.md
}

type sources struct {
	branch, head string
	commits      []string
	dirty        []string
	hasBeads     bool
	beadsProblem string     // why .beads exists but its tasks cannot be read here ("" when they can)
	snapshot     []snapTask // open tasks from .beads/issues.jsonl, used when the live ones cannot be read
	ready, doing []task
	blocked      []task
	hasSpecs     bool
	changes      []change
	specsProblem string // why openspec/ exists but its changes were listed from the files instead ("" when the tool worked)
	notes        []string
	harness      []string
	degraded     bool
	recentFiles  []string
	markers      []string
	teamHint     string // "" unless at least two signals of parallel work are present
}

func collect(dir string) sources {
	var s sources
	s.branch = strings.TrimSpace(must(run(dir, "git", "branch", "--show-current")))
	s.head = strings.TrimSpace(must(run(dir, "git", "log", "-1", "--format=%h %cs")))
	s.commits = lines(must(run(dir, "git", "log", "-20", "--format=%h %cs %s")))
	s.dirty = lines(must(run(dir, "git", "status", "--short")))

	if exists(filepath.Join(dir, ".beads")) {
		if !onPath("bd") {
			s.beadsProblem = "the bd command is not installed"
		} else if out, err := run(dir, "bd", "ready", "--json"); err != nil {
			s.beadsProblem = beadsFailure(err)
		} else {
			if err := json.Unmarshal([]byte(out), &s.ready); err != nil && strings.TrimSpace(out) != "" {
				// Exit code 0 but not the JSON we asked for: say so, never show "Ready: 0".
				s.beadsProblem = "bd returned output that could not be read"
			} else {
				s.hasBeads = true
				_ = json.Unmarshal([]byte(must(run(dir, "bd", "list", "--status", "in_progress", "--json"))), &s.doing)
				_ = json.Unmarshal([]byte(must(run(dir, "bd", "blocked", "--json"))), &s.blocked)
			}
		}
		if s.beadsProblem != "" {
			s.snapshot = readSnapshot(dir)
		}
	}
	if exists(filepath.Join(dir, "openspec")) {
		var out struct {
			Changes []change `json:"changes"`
		}
		switch {
		case !onPath("openspec"):
			s.specsProblem = "the openspec command is not installed"
		default:
			raw, err := run(dir, "openspec", "list", "--json")
			if err != nil {
				s.specsProblem = "openspec failed to list the changes"
			} else if jerr := json.Unmarshal([]byte(raw), &out); jerr != nil {
				s.specsProblem = "openspec returned output that could not be read"
			} else {
				s.hasSpecs = true
			}
		}
		if s.specsProblem != "" {
			out.Changes = specsOnDisk(dir) // same facts, read from the files, without the tool
		}
		s.changes = out.Changes
		for i, c := range s.changes {
			s.changes[i].Next = nextTask(filepath.Join(dir, "openspec", "changes", c.Name, "tasks.md"))
		}
	}
	s.teamHint = teamHint(ownerCount(dir), len(dedupe(lines(must(run(dir, "git", "log", "--since=14.days", "--format=%an"))))), assigneeCount(s.doing))
	inv := discover.Scan(dir)
	s.harness = inv.Lines(8)
	s.degraded = !inv.HasCoreHarness()
	if s.degraded {
		s.recentFiles = dedupe(lines(must(run(dir, "git", "log", "--since=7.days", "--name-only", "--pretty=format:"))))
		s.markers = lines(must(run(dir, "git", "grep", "-n", "-I", "-E", "TODO|FIXME")))
	}
	if b, err := os.ReadFile(filepath.Join(dir, ".relai", "notes.md")); err == nil {
		s.notes = strings.Split(strings.TrimRight(string(b), "\n"), "\n")
	}
	return s
}

// render builds the document. noteLines < 0 means "all notes".
func render(name string, s sources, commits, tasks, harness, recentCap, markersCap, noteLines int) string {
	var b strings.Builder
	p := func(f string, a ...any) { b.WriteString(cleanControl(fmt.Sprintf(f, a...)) + "\n") }

	p("# Handoff — %s", name)
	p("")
	degradedNote := ""
	if s.degraded {
		degradedNote = ", degraded mode — no AGENTS.md/CLAUDE.md, OpenSpec, or Beads found"
	}
	p("Generated by `relai handoff` (format v%s%s). Read this first; load anything else on demand.", FormatVersion, degradedNote)
	p("")
	p("## State")
	p("Branch `%s` at %s. Uncommitted files: %d.", or(s.branch, "?"), or(s.head, "no commits"), len(s.dirty))
	for i, d := range s.dirty {
		if i == 5 {
			p("  - … (%d more)", len(s.dirty)-5)
			break
		}
		if len(d) > 3 {
			d = d[3:]
		}
		p("  - `%s`", d)
	}
	p("")
	p("Recent commits:")
	for _, c := range head(s.commits, commits) {
		p("- %s", short(c, 110))
	}
	if len(s.commits) == 0 {
		p("- (none)")
	}
	if s.teamHint != "" {
		p("")
		p("%s", s.teamHint)
	}

	if !s.degraded {
		p("")
		p("## Tasks")
		if !s.hasBeads && s.beadsProblem != "" {
			p("Tasks cannot be read here: %s. Fresh clone? Run `bd bootstrap --yes` to load them from `.beads/issues.jsonl`.", s.beadsProblem)
			if len(s.snapshot) > 0 {
				p("From `.beads/issues.jsonl` (a snapshot kept in git, it may lag the live list):")
				var doing, rest []snapTask
				for _, t := range s.snapshot {
					if t.Status == "in_progress" {
						doing = append(doing, t)
					} else {
						rest = append(rest, t)
					}
				}
				if len(doing) > 0 {
					p("In progress:")
					for _, t := range head(doing, tasks) {
						p("- `%s` P%d %s", t.ID, t.Priority, short(t.Title, 100))
					}
				}
				p("Open (%d total):", len(rest))
				for _, t := range head(rest, tasks) {
					p("- `%s` P%d %s", t.ID, t.Priority, short(t.Title, 100))
				}
			}
		} else if !s.hasBeads {
			p("(no task graph in this repo)")
		} else {
			if stale := namedInCommits(s); len(stale) > 0 {
				p("Check first — still open but named in recent commits (done? close them):")
				for _, l := range head(stale, 5) {
					p("- %s", short(l, 110))
				}
			}
			if len(s.doing) > 0 {
				p("In progress:")
				for _, t := range head(s.doing, tasks) {
					p("- `%s` P%d %s", t.ID, t.Priority, short(t.Title, 100))
				}
			}
			p("Ready (unblocked, %d total):", len(s.ready))
			for _, t := range head(s.ready, tasks) {
				p("- `%s` P%d %s", t.ID, t.Priority, short(t.Title, 100))
			}
			if len(s.blocked) > 0 {
				p("Blocked (%d total):", len(s.blocked))
				for _, t := range head(s.blocked, tasks) {
					p("- `%s` P%d %s — blocked by %s", t.ID, t.Priority, short(t.Title, 80), "`"+strings.Join(t.BlockedBy, "`, `")+"`")
				}
			}
			p("Details: `bd show <id>`.")
		}

		p("")
		p("## Active changes (specs)")
		if s.specsProblem != "" {
			p("(%s: changes below are read from `openspec/changes/*/tasks.md`)", s.specsProblem)
		}
		switch {
		case !s.hasSpecs && s.specsProblem == "":
			p("(no OpenSpec changes in this repo)")
		case len(s.changes) == 0:
			p("No active change.")
		default:
			// The list is capped like the task lists: a repo with hundreds of changes must not
			// blow the budget. The first changes are the ones shown, the rest is a count.
			for _, c := range head(s.changes, tasks) {
				p("- `%s` %d/%d tasks (%s)", c.Name, c.CompletedTasks, c.TotalTasks, c.Status)
				if c.Next != "" {
					p("  next: %s", short(c.Next, 100))
				}
			}
			if len(s.changes) > tasks {
				p("- … (%d more changes)", len(s.changes)-tasks)
			}
			p("Details: `openspec show <name>`.")
		}
	}

	if s.degraded {
		p("")
		p("## Recently touched files (last 7 days)")
		if len(s.recentFiles) == 0 {
			p("(none)")
		} else {
			for _, f := range head(s.recentFiles, recentCap) {
				p("- `%s`", f)
			}
			if recentCap < len(s.recentFiles) {
				p("- … (%d more)", len(s.recentFiles)-recentCap)
			}
		}

		p("")
		p("## Open markers (TODO/FIXME)")
		if len(s.markers) == 0 {
			p("(none found)")
		} else {
			for _, m := range head(s.markers, markersCap) {
				p("- `%s`", m)
			}
			if markersCap < len(s.markers) {
				p("- … (%d more)", len(s.markers)-markersCap)
			}
		}
	}

	p("")
	p("## Harness")
	if len(s.harness) == 0 {
		p("(no agent harness found)")
	} else {
		for _, l := range head(s.harness, harness) {
			p("- %s", short(l, 140))
		}
		if harness < len(s.harness) {
			p("- … (%d more kinds)", len(s.harness)-harness)
		}
		p("Full inventory: `relai discover`.")
	}

	p("")
	p("## Decisions and pitfalls")
	if s.notes == nil {
		p("(no `.relai/notes.md` yet — run `relai discover --prompt` to draft it)")
	} else {
		n := s.notes
		if noteLines >= 0 && noteLines < len(n) {
			// Copy: appending to a prefix of s.notes would overwrite s.notes[noteLines] in the
			// shared array, corrupting later renders that show more lines.
			n = append(append([]string(nil), head(n, noteLines)...), fmt.Sprintf("… (%d more lines in `.relai/notes.md`)", len(s.notes)-noteLines))
		}
		for _, l := range n {
			p("%s", l)
		}
	}
	return b.String()
}

// TeamURL is where the optional team layer lives.
const TeamURL = "https://github.com/ECG-Rudy-Glt/RELAI-TEAM"

// teamHint returns the one-line suggestion when at least two independent signals of parallel work
// are present, else "". One signal alone is common in solo work (several owners listed, one person
// committing under several names), so it is not enough.
func teamHint(owners, authors, assignees int) string {
	var parts []string
	if owners >= 2 {
		parts = append(parts, fmt.Sprintf("%d owners", owners))
	}
	if authors >= 2 {
		parts = append(parts, fmt.Sprintf("%d commit authors (14 d)", authors))
	}
	if assignees >= 2 {
		parts = append(parts, fmt.Sprintf("%d assignees on in-progress tasks", assignees))
	}
	if len(parts) < 2 {
		return ""
	}
	return fmt.Sprintf("Team signals: %s — if people or agents work in parallel, relai-team adds live claim/release across repos: %s", strings.Join(parts, ", "), TeamURL)
}

func ownerCount(dir string) int {
	b, err := os.ReadFile(filepath.Join(dir, "relai.yaml"))
	if err != nil {
		return 0
	}
	return len(index.ParseManifest(b).Owners)
}

func assigneeCount(doing []task) int {
	seen := map[string]bool{}
	for _, t := range doing {
		if t.Assignee != "" {
			seen[t.Assignee] = true
		}
	}
	return len(seen)
}

// snapTask is one open task read from the JSONL export Beads keeps in git.
type snapTask struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
	Type     string `json:"issue_type"`
}

// readSnapshot reads the unfinished, non-epic tasks of .beads/issues.jsonl, most urgent first. It
// is the fallback for a clone whose Beads database has not been created yet, or a machine without bd.
func readSnapshot(dir string) []snapTask {
	b, err := os.ReadFile(filepath.Join(dir, ".beads", "issues.jsonl"))
	if err != nil {
		return nil
	}
	var out []snapTask
	for _, l := range strings.Split(string(b), "\n") {
		var t snapTask
		if json.Unmarshal([]byte(l), &t) != nil || t.ID == "" {
			continue
		}
		switch t.Status {
		case "open", "in_progress", "blocked":
			if t.Type != "epic" {
				out = append(out, t)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// beadsFailure turns a failed `bd` run into a short reason, without echoing raw tool output.
func beadsFailure(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) && strings.Contains(strings.ToLower(string(ee.Stderr)), "no beads database") {
		return "this clone has no Beads database yet"
	}
	return "bd failed to read the tasks"
}

// namedInCommits lists open tasks whose ID appears in a recent commit subject: often work that was
// merged without closing the task, which a stale task list would otherwise present as to do.
func namedInCommits(s sources) []string {
	var out []string
	seen := map[string]bool{}
	for _, group := range [][]task{s.doing, s.ready, s.blocked} {
		for _, t := range group {
			if seen[t.ID] {
				continue
			}
			seen[t.ID] = true
			re := regexp.MustCompile(`(^|[^\w-])` + regexp.QuoteMeta(t.ID) + `($|[^\w-])`)
			for _, c := range s.commits {
				if re.MatchString(c) {
					out = append(out, fmt.Sprintf("`%s` %s, in %s", t.ID, t.Title, c))
					break
				}
			}
		}
	}
	return out
}

// specsOnDisk lists the active OpenSpec changes by reading openspec/changes/*/tasks.md, for when
// the openspec command is missing or broken. Archived changes are skipped.
func specsOnDisk(dir string) []change {
	entries, err := os.ReadDir(filepath.Join(dir, "openspec", "changes"))
	if err != nil {
		return nil
	}
	var out []change
	for _, e := range entries {
		if !e.IsDir() || e.Name() == "archive" {
			continue
		}
		c := change{Name: e.Name(), Status: "from files"}
		if b, err := os.ReadFile(filepath.Join(dir, "openspec", "changes", e.Name(), "tasks.md")); err == nil {
			for _, l := range strings.Split(string(b), "\n") {
				l = strings.TrimSpace(l)
				switch {
				case strings.HasPrefix(l, "- [x] "), strings.HasPrefix(l, "- [X] "):
					c.CompletedTasks++
					c.TotalTasks++
				case strings.HasPrefix(l, "- [ ] "):
					c.TotalTasks++
				}
			}
		}
		out = append(out, c)
	}
	return out
}

// nextTask returns the first unchecked item ("- [ ] …") of an OpenSpec tasks.md, or "".
func nextTask(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); strings.HasPrefix(l, "- [ ] ") {
			return strings.TrimSpace(l[len("- [ ] "):])
		}
	}
	return ""
}

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
