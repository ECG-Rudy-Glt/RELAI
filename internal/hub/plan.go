package hub

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/validate"
)

// Kinds of action, which is what `relai check` reports drift on (ci is never drift: it is installed once).
const (
	KindConventions = "conventions"
	KindBlock       = "block"
	KindSkill       = "skill"
	KindCI          = "ci"
)

// ConventionsFile is where the hub's CONVENTIONS.md is vendored.
const ConventionsFile = ".relai/conventions.md"

// SkillRoots are the folders that receive the hub's skills, as hub-<name>/.
var SkillRoots = []string{".claude/skills", ".agents/skills"}

// Action is one change to the repository. Paths are relative to the repository root, slash-separated.
type Action struct {
	Op   string // "write" or "remove"
	Kind string
	Path string
	Data []byte
	Exec bool
	Why  string // "missing", "differs" or "stale"
}

// Plan is what `relai sync` would do. Building it changes nothing.
type Plan struct {
	Actions []Action
	Notes   []string // things to know or to do by hand
	Skipped []string // what was left alone, and why (symbolic links, big files, bad names)
}

func conventionsFile(pin Pin, c *Content) []byte {
	var b bytes.Buffer
	fmt.Fprintf(&b, "<!-- relai:generated %s — written by `relai sync`, do not edit: change the hub -->\n", pin)
	b.Write(c.Conventions)
	if len(c.Conventions) > 0 && c.Conventions[len(c.Conventions)-1] != '\n' {
		b.WriteByte('\n')
	}
	return b.Bytes()
}

const (
	blockStart = "<!-- relai:conventions"
	blockEnd   = "<!-- /relai:conventions -->"
)

// BlockLines is the managed block of AGENTS.md: three lines.
func BlockLines(pin Pin) []string {
	return []string{
		blockStart + " " + pin.String() + " — managed by `relai sync`, edit the hub instead -->",
		"Shared conventions of the hub: [" + ConventionsFile + "](" + ConventionsFile + "). The local rules of this file come after them and win.",
		blockEnd,
	}
}

// SetBlock returns text with the managed block set to pin, and whether a block was already there.
// Everything outside the markers is kept byte for byte. A new block goes before the first "## "
// heading (outside a code fence), or at the end. A start marker without its end marker is an error:
// guessing where the block ends could delete the author's text.
func SetBlock(text string, pin Pin) (string, bool, error) {
	lines := strings.SplitAfter(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	block := BlockLines(pin)
	start, end := -1, -1
	inFence := false
	first := -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~"):
			inFence = !inFence
		case start < 0 && strings.HasPrefix(t, blockStart):
			start = i
		case start >= 0 && end < 0 && t == blockEnd:
			end = i
		case !inFence && first < 0 && strings.HasPrefix(l, "## "):
			first = i
		}
	}
	if start >= 0 && end < 0 {
		return "", true, errors.New("AGENTS.md has a `<!-- relai:conventions` start marker without its `<!-- /relai:conventions -->` end marker: fix it by hand")
	}
	var b strings.Builder
	if start >= 0 {
		for _, l := range lines[:start] {
			b.WriteString(l)
		}
		for i, bl := range block {
			b.WriteString(bl)
			if i < len(block)-1 || strings.HasSuffix(lines[end], "\n") {
				b.WriteByte('\n')
			}
		}
		for _, l := range lines[end+1:] {
			b.WriteString(l)
		}
		return b.String(), true, nil
	}
	at := len(lines)
	if first >= 0 {
		at = first
	}
	for _, l := range lines[:at] {
		b.WriteString(l)
	}
	if at > 0 {
		if !strings.HasSuffix(lines[at-1], "\n") {
			b.WriteByte('\n')
		}
		if strings.TrimSpace(lines[at-1]) != "" {
			b.WriteByte('\n')
		}
	}
	for _, bl := range block {
		b.WriteString(bl + "\n")
	}
	if at < len(lines) {
		b.WriteByte('\n')
		for _, l := range lines[at:] {
			b.WriteString(l)
		}
	}
	return b.String(), false, nil
}

// Overrides returns the lines of text that start with "Override:" (after any indentation or list
// marker): the places where a repository declares that it departs from a hub rule.
func Overrides(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		t := strings.TrimSpace(l)
		t = strings.TrimSpace(strings.TrimLeft(t, "-*"))
		if strings.HasPrefix(t, "Override:") {
			out = append(out, t)
		}
	}
	return out
}

// within reports whether path, with symbolic links resolved, lies inside root.
func within(root, path string) bool {
	r, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	p, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(r, p)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// BuildPlan compares the repository at root with the hub content and returns what `relai sync`
// would change. It reads the repository and writes nothing.
func BuildPlan(root string, pin Pin, c *Content) (*Plan, error) {
	p := &Plan{}
	p.Skipped = append(p.Skipped, c.Skipped...)
	add := func(a Action) { p.Actions = append(p.Actions, a) }

	// 1. The vendored conventions file.
	want := conventionsFile(pin, c)
	switch got, state := readLocal(root, ConventionsFile); state {
	case localLink:
		p.Skipped = append(p.Skipped, ConventionsFile+": is a symbolic link, left alone")
	case localMissing:
		add(Action{Op: "write", Kind: KindConventions, Path: ConventionsFile, Data: want, Why: "missing"})
	case localFile:
		if !bytes.Equal(got, want) {
			add(Action{Op: "write", Kind: KindConventions, Path: ConventionsFile, Data: want, Why: "differs"})
		}
	default:
		p.Skipped = append(p.Skipped, ConventionsFile+": not a regular file, left alone")
	}

	// 2. The managed block of AGENTS.md (never created; written through a symbolic link that stays inside).
	agents := filepath.Join(root, "AGENTS.md")
	switch st, err := os.Lstat(agents); {
	case err != nil:
		p.Notes = append(p.Notes, "AGENTS.md not found: the managed block was not added (relai init writes one; AGENTS.md is never created here)")
	case st.Mode()&fs.ModeSymlink != 0 && !within(root, agents):
		p.Skipped = append(p.Skipped, "AGENTS.md: a symbolic link that leaves the repository, left alone")
	default:
		if st2, err := os.Stat(agents); err != nil || !st2.Mode().IsRegular() {
			p.Skipped = append(p.Skipped, "AGENTS.md: not a regular file, left alone")
			break
		}
		old, err := os.ReadFile(agents)
		if err != nil {
			return nil, fmt.Errorf("cannot read AGENTS.md: %w", err)
		}
		text, had, err := SetBlock(string(old), pin)
		if err != nil {
			return nil, err
		}
		if text != string(old) {
			why := "differs"
			if !had {
				why = "missing"
			}
			add(Action{Op: "write", Kind: KindBlock, Path: "AGENTS.md", Data: []byte(text), Why: why})
		}
	}

	// 3. The skills, in both folders.
	for _, sroot := range SkillRoots {
		if err := planSkills(root, sroot, c, p); err != nil {
			return nil, err
		}
	}

	// 4. The CI template, once, only where the repository already uses that system.
	planCI(root, c, p)

	sort.SliceStable(p.Actions, func(i, j int) bool { return p.Actions[i].Path < p.Actions[j].Path })
	return p, nil
}

type localState int

const (
	localMissing localState = iota
	localFile
	localLink
	localOther
)

func readLocal(root, rel string) ([]byte, localState) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	st, err := os.Lstat(path)
	switch {
	case err != nil:
		return nil, localMissing
	case st.Mode()&fs.ModeSymlink != 0:
		return nil, localLink
	case !st.Mode().IsRegular():
		return nil, localOther
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, localOther
	}
	return b, localFile
}

func planSkills(root, sroot string, c *Content, p *Plan) error {
	base := filepath.Join(root, filepath.FromSlash(sroot))
	st, err := os.Lstat(base)
	haveBase := err == nil
	if haveBase && (st.Mode()&fs.ModeSymlink != 0 || !st.IsDir()) {
		p.Skipped = append(p.Skipped, sroot+": not a plain folder, no skill installed there")
		return nil
	}
	wanted := map[string]bool{}
	for _, sk := range c.Skills {
		dname := "hub-" + sk.Name
		wanted[dname] = true
		dir := sroot + "/" + dname
		if lst, err := os.Lstat(filepath.Join(base, dname)); err == nil && (lst.Mode()&fs.ModeSymlink != 0 || !lst.IsDir()) {
			p.Skipped = append(p.Skipped, dir+": not a real folder, left alone")
			continue
		}
		local, err := localFiles(filepath.Join(base, dname))
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", dir, err)
		}
		hubFiles := map[string]bool{}
		for _, f := range sk.Files {
			hubFiles[f.Rel] = true
			path := dir + "/" + f.Rel
			l, ok := local[f.Rel]
			switch {
			case !ok:
				p.Actions = append(p.Actions, Action{Op: "write", Kind: KindSkill, Path: path, Data: f.Data, Exec: f.Exec, Why: "missing"})
			case l.link || !bytes.Equal(l.data, f.Data):
				if l.link { // a link where a file belongs: remove it first, then write the file
					p.Actions = append(p.Actions, Action{Op: "remove", Kind: KindSkill, Path: path, Why: "stale"})
				}
				p.Actions = append(p.Actions, Action{Op: "write", Kind: KindSkill, Path: path, Data: f.Data, Exec: f.Exec, Why: "differs"})
			}
		}
		for rel := range local {
			if !hubFiles[rel] {
				p.Actions = append(p.Actions, Action{Op: "remove", Kind: KindSkill, Path: dir + "/" + rel, Why: "stale"})
			}
		}
	}
	if !haveBase {
		return nil
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", sroot, err)
	}
	for _, e := range entries { // hub-* folders the hub no longer has
		name := e.Name()
		if !strings.HasPrefix(name, "hub-") || wanted[name] {
			continue
		}
		dir := sroot + "/" + name
		if e.Type()&fs.ModeSymlink != 0 || !e.IsDir() || !SafeName(name) {
			p.Skipped = append(p.Skipped, dir+": not a real folder or not a usable name, left alone")
			continue
		}
		local, err := localFiles(filepath.Join(base, name))
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", dir, err)
		}
		for rel := range local {
			p.Actions = append(p.Actions, Action{Op: "remove", Kind: KindSkill, Path: dir + "/" + rel, Why: "stale"})
		}
	}
	return nil
}

type localEntry struct {
	data []byte
	link bool
}

// localFiles lists the files under dir (nothing when it is absent), never following a link.
func localFiles(dir string) (map[string]localEntry, error) {
	out := map[string]localEntry{}
	if _, err := os.Lstat(dir); err != nil {
		return out, nil
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if d.Type()&fs.ModeSymlink != 0 {
			out[rel] = localEntry{link: true}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = localEntry{data: b}
		return nil
	})
	return out, err
}

func planCI(root string, c *Content, p *Plan) {
	exists := func(rel string) bool {
		_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
		return err == nil
	}
	install := func(kind, dir string) {
		dest := dir + "/workflows/relai.yml"
		if exists(dest) {
			return
		}
		data, ok := c.CI[kind]
		if !ok {
			p.Notes = append(p.Notes, fmt.Sprintf("the hub has no ci/%s.yml: no CI job added to %s", kind, dir))
			return
		}
		p.Actions = append(p.Actions, Action{Op: "write", Kind: KindCI, Path: dest, Data: data, Why: "missing"})
	}
	switch {
	case exists(".github"):
		install("github", ".github")
	case exists(".forgejo"):
		install("forgejo", ".forgejo")
	case exists(".gitea"):
		install("forgejo", ".gitea")
	}
	if exists(".gitlab-ci.yml") {
		if b, _ := os.ReadFile(filepath.Join(root, ".gitlab-ci.yml")); !bytes.Contains(b, []byte("relai check")) {
			if g, ok := c.CI["gitlab"]; ok {
				p.Notes = append(p.Notes, "GitLab: relai never edits .gitlab-ci.yml. Add these lines to it yourself:\n"+strings.TrimRight(string(g), "\n"))
			}
		}
	}
}

// Text renders the plan for a person. done says the actions were applied rather than only planned.
func (p *Plan) Text(done bool) string {
	var b strings.Builder
	verb := map[string]string{"write": "would write", "remove": "would remove"}
	if done {
		verb = map[string]string{"write": "wrote", "remove": "removed"}
	}
	for _, a := range p.Actions {
		fmt.Fprintf(&b, "%-12s %s (%s)\n", verb[a.Op], a.Path, a.Why)
	}
	for _, s := range p.Skipped {
		fmt.Fprintf(&b, "skipped      %s\n", s)
	}
	for _, n := range p.Notes {
		fmt.Fprintf(&b, "note         %s\n", n)
	}
	if len(p.Actions) == 0 {
		b.WriteString("up to date: nothing to change.\n")
	}
	return b.String()
}

// Drift turns the plan into the findings of `relai check`: everything the plan would write or
// remove, except the CI template, is vendored content that differs from the hub.
func (p *Plan) Drift(pin Pin) []validate.Finding {
	var out []validate.Finding
	for _, a := range p.Actions {
		if a.Kind == KindCI {
			continue
		}
		var msg string
		switch {
		case a.Op == "remove":
			msg = fmt.Sprintf("no longer in the hub at %s: run `relai sync --apply` to remove it", pin.Tag)
		case a.Why == "missing":
			msg = fmt.Sprintf("not vendored yet from the hub at %s: run `relai sync --apply`", pin.Tag)
		default:
			msg = fmt.Sprintf("differs from the hub at %s (a hand edit?): run `relai sync --apply` to restore it, or change the hub", pin.Tag)
		}
		if a.Kind == KindBlock {
			msg = "the managed conventions block " + msg
		}
		out = append(out, validate.Finding{Level: validate.Error, File: a.Path, Message: msg})
	}
	for _, s := range p.Skipped {
		out = append(out, validate.Finding{Level: validate.Info, File: "conventions", Message: "skipped: " + s})
	}
	return out
}
