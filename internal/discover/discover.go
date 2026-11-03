// Package discover finds what a project already gives its agents — instruction files, skills,
// commands, hooks, MCP servers, task tracking, specs, decision records, handoff and journal docs,
// CI — so that Relai can be adopted in the middle of a project without losing any of it.
//
// It only looks at file names and a few JSON keys; it never runs an LLM or a network call.
package discover

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Group is one kind of harness element, e.g. "Skills (.claude/skills)": deploy-playbook, …
type Group struct {
	Kind  string   `json:"kind"`
	Where string   `json:"where,omitempty"`
	Items []string `json:"items"`
}

// Inventory is everything found, in a fixed order.
type Inventory struct {
	Groups []Group `json:"groups"`
}

// Get returns the group of a kind, or nil.
func (inv Inventory) Get(kind string) *Group {
	for i := range inv.Groups {
		if inv.Groups[i].Kind == kind {
			return &inv.Groups[i]
		}
	}
	return nil
}

// Kinds, in display order.
const (
	Instructions = "Instructions"
	Skills       = "Skills"
	Commands     = "Commands"
	Subagents    = "Subagents"
	Hooks        = "Hooks"
	MCP          = "MCP servers"
	Tasks        = "Task tracking"
	Specs        = "Specs"
	Decisions    = "Decision records"
	Docs         = "Docs to read on demand"
	CI           = "CI"
)

var instructionFiles = []string{
	"AGENTS.md", "CLAUDE.md", "GEMINI.md", "CONVENTIONS.md", ".cursorrules", ".windsurfrules",
	".github/copilot-instructions.md",
}

// docPattern matches handoff, journal and plan documents worth pointing at; docDir matches the
// directories that hold such documents (a code directory named "handoff" is not one).
var (
	docPattern = regexp.MustCompile(`(?i)(handoff|passation|journal|roadmap|todo|changelog|runbook|onboarding)`)
	docDir     = regexp.MustCompile(`(?i)^(journals?|logbook|handoffs?|passations?|runbooks?)$`)
)

// Scan inventories the project at dir.
func Scan(dir string) Inventory {
	var inv Inventory
	add := func(kind, where string, items []string) {
		if len(items) > 0 {
			sort.Strings(items)
			inv.Groups = append(inv.Groups, Group{kind, where, dedupe(items)})
		}
	}

	var instr []string
	for _, f := range instructionFiles {
		if isFile(filepath.Join(dir, f)) {
			instr = append(instr, f)
		}
	}
	instr = append(instr, globRel(dir, ".cursor/rules/*.mdc")...)
	add(Instructions, "", instr)

	for _, root := range []string{".claude/skills", ".agents/skills", "skills"} {
		add(Skills, root, parentNames(dir, root+"/*/SKILL.md"))
	}
	var cmds []string
	for _, f := range globRel(dir, ".claude/commands/*.md") {
		cmds = append(cmds, "/"+strings.TrimSuffix(filepath.Base(f), ".md"))
	}
	for _, f := range globRel(dir, ".claude/commands/*/*.md") {
		cmds = append(cmds, "/"+filepath.Base(filepath.Dir(f))+":"+strings.TrimSuffix(filepath.Base(f), ".md"))
	}
	add(Commands, ".claude/commands", cmds)
	add(Subagents, ".claude/agents", baseNames(globRel(dir, ".claude/agents/*.md"), ".md"))

	for _, f := range []string{".claude/settings.json", ".codex/hooks.json"} {
		add(Hooks, f, jsonKeys(filepath.Join(dir, f), "hooks"))
	}
	add(MCP, ".mcp.json", jsonKeys(filepath.Join(dir, ".mcp.json"), "mcpServers"))

	var tasks []string
	if isDir(filepath.Join(dir, ".beads")) {
		tasks = append(tasks, "Beads (.beads, `bd ready`)")
	}
	if url := remoteURL(dir); strings.Contains(url, "github.com") {
		tasks = append(tasks, "GitHub Issues (`gh issue list`)")
	}
	add(Tasks, "", tasks)

	var specs []string
	if isDir(filepath.Join(dir, "openspec")) {
		specs = append(specs, "OpenSpec (openspec/, `openspec list`)")
	}
	if isDir(filepath.Join(dir, ".specify")) {
		specs = append(specs, "Spec Kit (.specify/, specs/)")
	}
	add(Specs, "", specs)

	var adrs, docs []string
	walk(dir, 3, func(rel string, d fs.DirEntry) {
		name := d.Name()
		if d.IsDir() {
			if l := strings.ToLower(name); l == "adr" || l == "adrs" || l == "decisions" {
				adrs = append(adrs, rel+"/")
			} else if docDir.MatchString(name) && len(globRel(filepath.Join(dir, rel), "*.md")) > 0 {
				docs = append(docs, rel+"/")
			}
			return
		}
		if strings.HasSuffix(strings.ToLower(name), ".md") && docPattern.MatchString(name) {
			docs = append(docs, rel)
		}
	})
	add(Decisions, "", adrs)
	add(Docs, "", collapse(docs))

	var ci []string
	for _, g := range []string{".github/workflows/*.y*ml", ".forgejo/workflows/*.y*ml", ".gitea/workflows/*.y*ml"} {
		ci = append(ci, globRel(dir, g)...)
	}
	if isFile(filepath.Join(dir, ".gitlab-ci.yml")) {
		ci = append(ci, ".gitlab-ci.yml")
	}
	add(CI, "", ci)
	return inv
}

// Lines renders one compact line per group, for the handoff's Harness section.
func (inv Inventory) Lines(maxItems int) []string {
	var out []string
	for _, g := range inv.Groups {
		label := g.Kind
		if g.Where != "" {
			label += " (" + g.Where + ")"
		}
		items := g.Items
		more := ""
		if len(items) > maxItems {
			more = ", … (" + strconv.Itoa(len(items)-maxItems) + " more)"
			items = items[:maxItems]
		}
		out = append(out, label+": "+strings.Join(items, ", ")+more)
	}
	return out
}

// walk visits files and directories up to depth, skipping VCS, dependency and hidden directories
// (except the ones the harness lives in, which Scan reads directly).
func walk(dir string, depth int, fn func(rel string, d fs.DirEntry)) {
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == dir {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		if d.IsDir() {
			n := d.Name()
			if strings.HasPrefix(n, ".") || n == "node_modules" || n == "vendor" || n == "archive" {
				return filepath.SkipDir
			}
			if strings.Count(rel, string(filepath.Separator)) >= depth {
				return filepath.SkipDir
			}
		}
		fn(filepath.ToSlash(rel), d)
		return nil
	})
}

// collapse keeps a directory instead of the many files under it, so the list stays short.
func collapse(paths []string) []string {
	dirs := map[string]bool{}
	for _, p := range paths {
		if strings.HasSuffix(p, "/") {
			dirs[p] = true
		}
	}
	var out []string
	for _, p := range paths {
		inDir := false
		for d := range dirs {
			if p != d && strings.HasPrefix(p, d) {
				inDir = true
				break
			}
		}
		if !inDir {
			out = append(out, p)
		}
	}
	return out
}

func globRel(dir, pattern string) []string {
	m, _ := filepath.Glob(filepath.Join(dir, pattern))
	var out []string
	for _, p := range m {
		rel, _ := filepath.Rel(dir, p)
		out = append(out, filepath.ToSlash(rel))
	}
	return out
}

func parentNames(dir, pattern string) []string {
	var out []string
	for _, p := range globRel(dir, pattern) {
		out = append(out, filepath.Base(filepath.Dir(p)))
	}
	return out
}

func baseNames(paths []string, ext string) []string {
	var out []string
	for _, p := range paths {
		out = append(out, strings.TrimSuffix(filepath.Base(p), ext))
	}
	return out
}

// jsonKeys returns the keys of an object field in a JSON file, e.g. the hook events of
// .claude/settings.json or the servers of .mcp.json.
func jsonKeys(path, field string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc map[string]json.RawMessage
	if json.Unmarshal(b, &doc) != nil {
		return nil
	}
	var obj map[string]json.RawMessage
	if json.Unmarshal(doc[field], &obj) != nil {
		return nil
	}
	var out []string
	for k := range obj {
		out = append(out, k)
	}
	return out
}

func remoteURL(dir string) string {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func dedupe(s []string) []string {
	var out []string
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

func isFile(p string) bool { st, err := os.Stat(p); return err == nil && !st.IsDir() }
func isDir(p string) bool  { st, err := os.Stat(p); return err == nil && st.IsDir() }
