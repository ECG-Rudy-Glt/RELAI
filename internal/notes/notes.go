// Package notes keeps .relai/notes.md, the project's curated memory, in shape: add a dated entry
// under the right heading, list the old ones, move them to an archive. No LLM, nothing rewritten:
// lines are added or moved whole.
package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// File and ArchiveFile are relative to the repository root.
const (
	File        = ".relai/notes.md"
	ArchiveFile = ".relai/notes-archive.md"
)

const dateLayout = "2006-01-02"

var (
	headingPattern = regexp.MustCompile(`^#{1,6}\s+(.*\S)\s*$`)
	datedPattern   = regexp.MustCompile(`^- (\d{4}-\d{2}-\d{2})\b`)
)

// Kinds of entries `relai note add` knows where to put.
const (
	Decision = "Decisions"
	Pitfall  = "Pitfalls"
	Next     = "Next session"
)

// Add inserts "- <date> — <text>" at the end of the section named kind in dir's notes file,
// creating the file or the section when missing. It returns the line it wrote.
func Add(dir, kind, text string, now time.Time) (string, error) {
	text = strings.Join(strings.Fields(text), " ") // one line, whatever was pasted
	if text == "" {
		return "", fmt.Errorf("nothing to add: give the text of the note")
	}
	entry := fmt.Sprintf("- %s — %s", now.Format(dateLayout), text)
	path := filepath.Join(dir, File)
	b, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	lines := splitLines(string(b))
	start := -1
	for i, l := range lines {
		if m := headingPattern.FindStringSubmatch(l); m != nil && strings.EqualFold(m[1], kind) {
			start = i
			break
		}
	}
	if start < 0 {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "### "+kind, entry)
	} else {
		end := start + 1 // one past the section's last non-blank line
		for i := start + 1; i < len(lines) && !headingPattern.MatchString(lines[i]); i++ {
			if strings.TrimSpace(lines[i]) != "" {
				end = i + 1
			}
		}
		lines = append(lines[:end:end], append([]string{entry}, lines[end:]...)...)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return entry, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

// Entry is one dated entry of the notes: its first line and the indented lines that continue it.
type Entry struct {
	Section    string
	Date       time.Time
	First, End int // lines[First:End]
	Text       string
}

// Report is what Check found.
type Report struct {
	Lines, Tokens int     // size of the notes file (tokens estimated as in the handoff: characters / 4)
	Dated         int     // dated entries in all
	Old           []Entry // those older than the limit, in file order
}

// Check reads dir's notes and lists the dated entries older than days, optionally only in the
// sections whose heading contains section (case-insensitive).
func Check(dir string, days int, section string, now time.Time) (Report, error) {
	b, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		return Report{}, fmt.Errorf("no %s here (relai init writes one)", File)
	}
	lines := splitLines(string(b))
	r := Report{Lines: len(lines), Tokens: (len([]rune(string(b))) + 3) / 4}
	limit := now.AddDate(0, 0, -days)
	for _, e := range entries(lines) {
		r.Dated++
		if e.Date.Before(limit) && strings.Contains(strings.ToLower(e.Section), strings.ToLower(section)) {
			r.Old = append(r.Old, e)
		}
	}
	return r, nil
}

func entries(lines []string) []Entry {
	var out []Entry
	section := ""
	for i := 0; i < len(lines); i++ {
		if m := headingPattern.FindStringSubmatch(lines[i]); m != nil {
			section = m[1]
			continue
		}
		m := datedPattern.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		d, err := time.Parse(dateLayout, m[1])
		if err != nil {
			continue
		}
		end := i + 1
		for end < len(lines) && (strings.HasPrefix(lines[end], " ") || strings.HasPrefix(lines[end], "\t")) && strings.TrimSpace(lines[end]) != "" {
			end++
		}
		out = append(out, Entry{Section: section, Date: d, First: i, End: end, Text: lines[i]})
		i = end - 1
	}
	return out
}

// Archive moves the entries Check lists from the notes to the archive file, under a heading that
// names the section and the day. It returns how many entries moved. Nothing is deleted: the lines
// are appended to the archive before the notes are rewritten.
func Archive(dir string, days int, section string, now time.Time) (int, error) {
	r, err := Check(dir, days, section, now)
	if err != nil || len(r.Old) == 0 {
		return 0, err
	}
	path := filepath.Join(dir, File)
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	lines := splitLines(string(b))
	drop := make([]bool, len(lines))
	var moved strings.Builder
	last := ""
	for _, e := range r.Old {
		if e.Section != last {
			fmt.Fprintf(&moved, "\n### %s (archived %s)\n", e.Section, now.Format(dateLayout))
			last = e.Section
		}
		for i := e.First; i < e.End; i++ {
			drop[i] = true
			moved.WriteString(lines[i] + "\n")
		}
	}
	apath := filepath.Join(dir, ArchiveFile)
	old, err := os.ReadFile(apath)
	if err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	head := string(old)
	if head == "" {
		head = "# Archived notes\n\nEntries moved out of `notes.md` by `relai note archive`. The handoff does not read this file.\n"
	}
	if err := os.WriteFile(apath, []byte(strings.TrimRight(head, "\n")+"\n"+moved.String()), 0o644); err != nil {
		return 0, err
	}
	var kept []string
	for i, l := range lines {
		if !drop[i] {
			kept = append(kept, l)
		}
	}
	return len(r.Old), os.WriteFile(path, []byte(strings.Join(kept, "\n")+"\n"), 0o644)
}

// Text renders a report for a person. budget is the handoff's token budget.
func (r Report) Text(days, budget int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %d lines, about %d tokens (the whole handoff has %d), %d dated entries.\n", File, r.Lines, r.Tokens, budget, r.Dated)
	if r.Tokens > budget/2 {
		b.WriteString("The notes alone take more than half of the handoff: the end of the file is cut when the budget is short.\n")
	}
	if len(r.Old) == 0 {
		fmt.Fprintf(&b, "No dated entry is older than %d days.\n", days)
		return b.String()
	}
	fmt.Fprintf(&b, "%d entries are older than %d days. Still true? Keep them. Done or obsolete? `relai note archive --apply` moves them to %s.\n", len(r.Old), days, ArchiveFile)
	for _, e := range r.Old {
		t := e.Text
		if rs := []rune(t); len(rs) > 110 {
			t = string(rs[:109]) + "…"
		}
		fmt.Fprintf(&b, "  [%s] %s\n", e.Section, t)
	}
	return b.String()
}

func splitLines(s string) []string {
	s = strings.TrimRight(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
