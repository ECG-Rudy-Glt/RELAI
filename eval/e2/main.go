// Command e2 runs experiment E2: does an agent that starts cold with AGENTS.md + `relai handoff`
// pick the right next step as often as one that reads the docs, for far fewer tokens?
//
// For each fixture (eval/fixtures/<name>), condition and agent, it builds a fresh copy of the
// fixture repo, runs the agent headless with a fixed prompt, grades the "NEXT:" line of its answer
// against expected.json and records tokens, turns and cost. See eval/README.md.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
)

const task = `You are an AI agent joining this repository cold: you have no prior context.
Find out where the work stands and decide the single next step you would work on now.
Do not modify any file and do not start the work itself.
End your answer with exactly one line of the form:
NEXT: <task ID if the project has one> — <the next step in one sentence>`

// Preambles per condition: (a) nothing, (b) read the docs, (c) the Relai repo contract, with
// AGENTS.md and the handoff injected as a SessionStart hook would do.
var conditions = []string{"none", "docs", "relai"}

const docsPreamble = `Start by reading the project documentation (README, docs/, TODO and journal files) before deciding.

`

const agentsMD = `# Instructions for AI agents

Start here:
1. Read the handoff below (or run ` + "`relai handoff`" + `): state, next tasks, decisions, pitfalls.
2. Load anything else only when you need it.
`

type agent struct {
	name  string
	model string
}

var agents = map[string]agent{
	"claude-haiku":  {"claude-haiku", "haiku"},
	"claude-sonnet": {"claude-sonnet", "sonnet"},
	"claude-opus":   {"claude-opus", "opus"},
}

// Result is one line of results.jsonl.
type Result struct {
	Fixture   string  `json:"fixture"`
	Condition string  `json:"condition"`
	Agent     string  `json:"agent"`
	Run       int     `json:"run"`
	Next      string  `json:"next"`
	Correct   bool    `json:"correct"`
	Trap      string  `json:"trap,omitempty"`
	InputTok  int     `json:"input_tokens"` // input + cache read + cache creation, all turns
	OutputTok int     `json:"output_tokens"`
	Turns     int     `json:"turns"`
	CostUSD   float64 `json:"cost_usd"`
	Seconds   float64 `json:"seconds"`
	Error     string  `json:"error,omitempty"`
}

func main() {
	var (
		fixtures = flag.String("fixtures", "", "comma-separated fixture names (default: all)")
		repos    = flag.String("repos", "", "comma-separated paths of real repos to run instead of fixtures (ungraded)")
		conds    = flag.String("conditions", strings.Join(conditions, ","), "conditions to run")
		agentsF  = flag.String("agents", "claude-haiku,claude-sonnet", "agents to run")
		runs     = flag.Int("runs", 1, "runs per cell")
		par      = flag.Int("parallel", 3, "concurrent agent runs")
		out      = flag.String("out", "eval/results", "output directory")
		dry      = flag.Bool("dry", false, "build the fixtures and print the prompts, run no agent")
	)
	flag.Parse()
	if err := run(*fixtures, *repos, *conds, *agentsF, *runs, *par, *out, *dry); err != nil {
		fmt.Fprintln(os.Stderr, "e2:", err)
		os.Exit(1)
	}
}

func run(fixtureList, repoList, condList, agentList string, runs, par int, out string, dry bool) error {
	root, err := filepath.Abs("eval/fixtures")
	if err != nil {
		return err
	}
	names, err := fixtureNames(root, fixtureList)
	if err != nil {
		return err
	}
	if repoList != "" {
		names = nil
		for _, r := range split(repoList) {
			abs, err := filepath.Abs(r)
			if err != nil {
				return err
			}
			names = append(names, abs)
		}
	}
	// Paths must not leak the fixture or the condition to the agent: it sees its working directory.
	work, err := os.MkdirTemp("", "ws-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	// The agent may call `relai handoff` itself in condition (c).
	bin := filepath.Join(work, "bin")
	if out, err := exec.Command("go", "build", "-o", filepath.Join(bin, "relai"), "./cmd/relai").CombinedOutput(); err != nil {
		return fmt.Errorf("build relai: %v\n%s", err, out)
	}
	os.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	type job struct {
		fixture, cond, dir, prompt string
		exp                        Expected
		ag                         agent
		run                        int
	}
	var jobs []job
	for _, name := range names {
		label, exp := name, Expected{}
		if filepath.IsAbs(name) {
			label = "real:" + filepath.Base(name) // no known answer: graded by hand
		} else if exp, err = loadExpected(filepath.Join(root, name, "expected.json")); err != nil {
			return err
		}
		for _, cond := range split(condList) {
			for _, an := range split(agentList) {
				ag, ok := agents[an]
				if !ok {
					return fmt.Errorf("unknown agent %q", an)
				}
				for r := 1; r <= runs; r++ {
					dir := filepath.Join(work, fmt.Sprintf("%03d", len(jobs)), "project")
					prompt, err := prepare(root, name, cond, dir)
					if err != nil {
						return fmt.Errorf("%s/%s: %v", label, cond, err)
					}
					jobs = append(jobs, job{label, cond, dir, prompt, exp, ag, r})
				}
			}
		}
	}
	if dry {
		seen := map[string]bool{}
		for _, j := range jobs {
			if k := j.fixture + "/" + j.cond; !seen[k] {
				seen[k] = true
				fmt.Printf("===== %s (%s) — %d tokens\n%s\n\n", k, j.dir, handoff.EstimateTokens(j.prompt), j.prompt)
			}
		}
		return nil
	}

	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format("20060102-150405")
	f, err := os.Create(filepath.Join(out, stamp+".jsonl"))
	if err != nil {
		return err
	}
	defer f.Close()

	var (
		mu      sync.Mutex
		results []Result
		wg      sync.WaitGroup
		sem     = make(chan struct{}, par)
	)
	for _, j := range jobs {
		wg.Add(1)
		sem <- struct{}{}
		go func(j job) {
			defer func() { <-sem; wg.Done() }()
			r := runClaude(j.ag, j.dir, j.prompt)
			r.Fixture, r.Condition, r.Agent, r.Run = j.fixture, j.cond, j.ag.name, j.run
			if r.Error == "" {
				r.Correct, r.Trap = j.exp.Grade(r.Next)
			}
			mu.Lock()
			defer mu.Unlock()
			results = append(results, r)
			b, _ := json.Marshal(r)
			f.Write(append(b, '\n'))
			fmt.Fprintf(os.Stderr, "%-22s %-6s %-14s #%d correct=%-5v in=%-7d turns=%-3d %s\n",
				r.Fixture, r.Condition, r.Agent, r.Run, r.Correct, r.InputTok, r.Turns, short(r.Next, 70)+r.Error)
		}(j)
	}
	wg.Wait()
	fmt.Print(Summary(results))
	return nil
}

// prepare builds the fixture (or clones the real repo at an absolute path) into dir and returns
// the prompt for the condition.
func prepare(root, name, cond, dir string) (string, error) {
	real := filepath.IsAbs(name)
	if real {
		if err := cloneRepo(name, dir); err != nil {
			return "", err
		}
	} else {
		cmd := exec.Command("bash", filepath.Join(root, name, "build.sh"), dir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("build.sh: %v\n%s", err, out)
		}
	}
	switch cond {
	case "none":
		return task, nil
	case "docs":
		return docsPreamble + task, nil
	case "relai":
		if real {
			// A real repo keeps its own AGENTS.md and notes, if any: this measures an uncurated
			// adoption, just the handoff.
			h, err := handoff.Build(dir, handoff.DefaultOptions())
			if err != nil {
				return "", err
			}
			return "Session start context (SessionStart hook):\n\n--- AGENTS.md ---\n" + agentsMD +
				"\n--- relai handoff ---\n" + h + "\n---\n\n" + task, nil
		}
		notes, err := os.ReadFile(filepath.Join(root, name, "notes.md"))
		if err != nil {
			return "", err
		}
		files := map[string]string{"AGENTS.md": agentsMD, ".relai/notes.md": string(notes)}
		for p, c := range files {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(dir, p), []byte(c), 0o644); err != nil {
				return "", err
			}
		}
		if out, err := gitIn(dir, "add", "AGENTS.md", ".relai"); err != nil {
			return "", fmt.Errorf("git add: %v %s", err, out)
		}
		if out, err := gitIn(dir, "-c", "user.name=Sam Doe", "-c", "user.email=sam@example.com",
			"commit", "-q", "-m", "chore: adopt the relai repo contract"); err != nil {
			return "", fmt.Errorf("git commit: %v %s", err, out)
		}
		h, err := handoff.Build(dir, handoff.DefaultOptions())
		if err != nil {
			return "", err
		}
		return "Session start context (SessionStart hook):\n\n--- AGENTS.md ---\n" + agentsMD +
			"\n--- relai handoff ---\n" + h + "\n---\n\n" + task, nil
	}
	return "", fmt.Errorf("unknown condition %q", cond)
}

// cloneRepo copies a real repo for an agent run: tracked files only (no untracked secrets), the
// Beads database, no remote (a stray `git push` must not reach the original) and no project
// settings (their hooks would add context the conditions do not control).
func cloneRepo(src, dir string) error {
	steps := [][]string{
		{"git", "clone", "-q", "--no-hardlinks", src, dir},
		{"git", "-C", dir, "remote", "remove", "origin"},
	}
	if exists(filepath.Join(src, ".beads")) {
		steps = append(steps, []string{"cp", "-a", filepath.Join(src, ".beads") + "/.", filepath.Join(dir, ".beads")})
	}
	for _, f := range []string{".claude/settings.json", ".claude/settings.local.json"} {
		if exists(filepath.Join(dir, f)) || exists(filepath.Join(src, f)) {
			steps = append(steps, []string{"git", "-C", dir, "update-index", "--skip-worktree", f},
				[]string{"rm", "-f", filepath.Join(dir, f)})
		}
	}
	for _, s := range steps {
		if out, err := exec.Command(s[0], s[1:]...).CombinedOutput(); err != nil && s[1] != "-C" {
			return fmt.Errorf("%v: %v %s", s, err, out)
		}
	}
	return nil
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

func gitIn(dir string, args ...string) ([]byte, error) {
	const when = "2026-06-30T10:00:00Z" // after every fixture commit, fixed for reproducibility
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when)
	return cmd.CombinedOutput()
}

// runClaude runs Claude Code headless, isolated from the user's settings, memory, hooks and MCP
// servers, with read-only tools.
func runClaude(ag agent, dir, prompt string) Result {
	start := time.Now()
	cmd := exec.Command("claude", "-p", prompt,
		"--model", ag.model,
		"--output-format", "json",
		"--setting-sources", "project",
		"--strict-mcp-config",
		"--no-session-persistence",
		"--allowedTools", "Read,Glob,Grep,Bash(git:*),Bash(bd:*),Bash(openspec:*),Bash(relai:*),Bash(ls:*),Bash(cat:*),Bash(head:*),Bash(find:*),Bash(wc:*)",
		"--disallowedTools", "Edit,Write,NotebookEdit,WebFetch,WebSearch,Agent,Task",
	)
	cmd.Dir = dir
	out, err := cmd.Output()
	r := Result{Seconds: time.Since(start).Seconds()}
	var resp struct {
		Result   string  `json:"result"`
		IsError  bool    `json:"is_error"`
		NumTurns int     `json:"num_turns"`
		Cost     float64 `json:"total_cost_usd"`
		Usage    struct {
			Input       int `json:"input_tokens"`
			CacheRead   int `json:"cache_read_input_tokens"`
			CacheCreate int `json:"cache_creation_input_tokens"`
			Output      int `json:"output_tokens"`
		} `json:"usage"`
	}
	if jerr := json.Unmarshal(out, &resp); jerr != nil {
		r.Error = fmt.Sprintf("agent: %v %v %s", err, jerr, short(string(out), 200))
		return r
	}
	u := resp.Usage
	r.InputTok = u.Input + u.CacheRead + u.CacheCreate
	r.OutputTok, r.Turns, r.CostUSD = u.Output, resp.NumTurns, resp.Cost
	r.Next = NextLine(resp.Result)
	if resp.IsError {
		r.Error = "agent error: " + short(resp.Result, 200)
	}
	return r
}

// Summary renders a Markdown table: one row per fixture × condition × agent, averaged over runs.
func Summary(rs []Result) string {
	type key struct{ f, c, a string }
	type acc struct {
		n, ok, ung  int
		in, turns   int
		cost        float64
		traps, errs []string
	}
	m := map[key]*acc{}
	var keys []key
	for _, r := range rs {
		k := key{r.Fixture, r.Condition, r.Agent}
		if m[k] == nil {
			m[k] = &acc{}
			keys = append(keys, k)
		}
		a := m[k]
		if r.Error != "" {
			a.errs = append(a.errs, r.Error)
			continue
		}
		a.n++
		a.in += r.InputTok
		a.turns += r.Turns
		a.cost += r.CostUSD
		switch {
		case r.Correct:
			a.ok++
		case r.Trap == ungraded:
			a.ung++
		case r.Trap != "":
			a.traps = append(a.traps, r.Trap)
		}
	}
	order := map[string]int{}
	for i, c := range conditions {
		order[c] = i
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.f != b.f {
			return a.f < b.f
		}
		if a.a != b.a {
			return a.a < b.a
		}
		return order[a.c] < order[b.c]
	})
	var b strings.Builder
	b.WriteString("\n| Fixture | Agent | Condition | Correct | Avg input tokens | Avg turns | Avg cost | Notes |\n|---|---|---|---|---|---|---|---|\n")
	for _, k := range keys {
		a := m[k]
		notes := strings.Join(append(a.traps, a.errs...), "; ")
		if a.n == 0 {
			fmt.Fprintf(&b, "| %s | %s | %s | – | – | – | – | %s |\n", k.f, k.a, k.c, notes)
			continue
		}
		correct := fmt.Sprintf("%d/%d", a.ok, a.n-a.ung)
		if a.ung == a.n {
			correct = "by hand"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %d | %.1f | $%.3f | %s |\n",
			k.f, k.a, k.c, correct, a.in/a.n, float64(a.turns)/float64(a.n), a.cost/float64(a.n), notes)
	}
	return b.String()
}

func fixtureNames(root, list string) ([]string, error) {
	if list != "" {
		return split(list), nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func split(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func short(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}
