// Command relai makes a repository agent-native: a small repo contract and a compact handoff.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/discover"
	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
	"github.com/ECG-Rudy-Glt/RELAI/internal/initrepo"
	"github.com/ECG-Rudy-Glt/RELAI/internal/scaffold"
	"github.com/ECG-Rudy-Glt/RELAI/internal/scan"
	"github.com/ECG-Rudy-Glt/RELAI/internal/tidy"
	"github.com/ECG-Rudy-Glt/RELAI/internal/workspace"
)

// defaultIndexPath is where `relai index build` writes and `relai find` reads by default.
const defaultIndexPath = ".relai/index.json"

var version = "dev" // set at build time: -ldflags "-X main.version=v0.1.0"

const usage = `relai — pass the baton between people and their AI agents

Usage:
  relai init [dir]       add AGENTS.md, relai.yaml, .relai/notes.md, and for Claude Code a CLAUDE.md and a
                         session-start hook that loads the handoff (never overwrites; --no-claude skips the last two;
                         --hooks adds a git pre-commit hook that keeps .beads/issues.jsonl current)
  relai handoff [dir]    print the current state of the work (≤ 2,000 tokens by default)
  relai discover [dir]   list what the project already gives agents (skills, hooks, specs, docs…);
                         --prompt prints instructions for your agent to draft .relai/notes.md
  relai index build [-o <path>] [--scan <folder>]... [--depth N] [<list-file>]
                         read each repo path/URL in <list-file> (and/or every git repo in each --scan <folder>;
                         repeat --scan to cover several folders, a repo reached by several of them is indexed once)
                         and collect their relai.yaml into an index (default: .relai/index.json)
  relai discover --links <folder> [--depth N] [--propose [--min-weak N] | --prompt] [--json]
                         mentions between the git repos of <folder> (strong: remote URL or relative path;
                         weak: the name as a word). Evidence, not verified links; --propose prints a
                         links: block per relai.yaml to check and paste yourself; --prompt prints instructions
                         for your agent to verify each mention with you (nothing is ever written)
  relai find [--index <path>] <capability>
                         search the index for repos that already provide <capability>
  relai workspace [--index <path>] [--max-tokens N]
                         one short overview of every project in the index: last activity, open tasks,
                         what each provides (reads files only, never runs bd; default 800 tokens)
  relai tidy [--apply] [dir]
                         after work is merged: close the tasks a commit says it closes ("Closes <id>"), refresh
                         .beads/issues.jsonl, delete fully merged local branches. Dry run unless --apply; never commits
  relai new <template> <dir>
                         scaffold <dir> from <template> (a local path or clone URL), substituting
                         {{.RepoName}} etc. in its text files, then filling in anything the
                         template didn't already provide (AGENTS.md, relai.yaml, .relai/notes.md)
  relai version

Run "relai <command> -h" for options.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		fs := flag.NewFlagSet("init", flag.ExitOnError)
		noLink := fs.Bool("no-link", false, "write a new AGENTS.md instead of linking it to an existing CLAUDE.md")
		noClaude := fs.Bool("no-claude", false, "do not add CLAUDE.md and the Claude Code session-start hook")
		hooks := fs.Bool("hooks", false, "also install a git pre-commit hook that keeps .beads/issues.jsonl current")
		_ = fs.Parse(reorderArgs(fs, os.Args[2:]))
		r, err := initrepo.InitWith(dirArg(fs), initrepo.Options{LinkClaude: !*noLink, ClaudeCode: !*noClaude, GitHook: *hooks})
		exitOn(err)
		for _, c := range r.Created {
			fmt.Println("created ", c)
		}
		for _, s := range r.Skipped {
			fmt.Println("kept    ", s)
		}
		if inv := discover.Scan(dirArg(fs)); len(inv.Groups) > 0 {
			fmt.Println("\nAlready in this project (the handoff will point to it):")
			for _, l := range inv.Lines(6) {
				fmt.Println("  - " + l)
			}
		}
		fmt.Println("\nNext:")
		fmt.Println("  1. write a one-line summary in relai.yaml;")
		fmt.Println("  2. have your agent draft .relai/notes.md from the existing docs: relai discover --prompt;")
		fmt.Println("  3. review the draft, commit, and run relai handoff.")
	case "handoff":
		o := handoff.DefaultOptions()
		fs := flag.NewFlagSet("handoff", flag.ExitOnError)
		fs.IntVar(&o.MaxTokens, "max-tokens", o.MaxTokens, "token budget for the whole handoff")
		out := fs.String("o", "", "write to this file instead of stdout (e.g. .relai/handoff.md)")
		_ = fs.Parse(reorderArgs(fs, os.Args[2:]))
		doc, err := handoff.Build(dirArg(fs), o)
		exitOn(err)
		if *out == "" {
			fmt.Print(doc)
			return
		}
		exitOn(os.WriteFile(*out, []byte(doc), 0o644))
		fmt.Fprintf(os.Stderr, "wrote %s (~%d tokens)\n", *out, handoff.EstimateTokens(doc))
	case "discover":
		fs := flag.NewFlagSet("discover", flag.ExitOnError)
		prompt := fs.Bool("prompt", false, "print instructions for an agent to draft .relai/notes.md from what was found")
		asJSON := fs.Bool("json", false, "print the inventory as JSON")
		links := fs.Bool("links", false, "treat the argument as a folder of repositories and print the mentions between them")
		propose := fs.Bool("propose", false, "with --links: print a links: block for each relai.yaml to check and paste yourself")
		depth := fs.Int("depth", 1, "with --links: how many folder levels to look down for repositories (1 = directly inside)")
		minWeak := fs.Int("min-weak", scan.DefaultMinWeak, "with --propose: files that must mention a name, in documents only, to propose it without strong evidence")
		_ = fs.Parse(reorderArgs(fs, os.Args[2:]))
		target := dirArg(fs)
		if st, err := os.Stat(target); err != nil || !st.IsDir() {
			fmt.Fprintf(os.Stderr, "relai: %s is not a directory\n", target)
			os.Exit(1)
		}
		if *links || *propose {
			repos, err := scan.FindDepth(target, *depth)
			exitOn(err)
			ls := scan.Links(repos)
			switch {
			case *asJSON:
				b, _ := json.MarshalIndent(map[string]any{"repos": repos, "links": ls}, "", "  ")
				fmt.Println(string(b))
			case *prompt:
				fmt.Print(scan.Prompt(target, repos, ls))
			case *propose:
				fmt.Print(scan.Propose(repos, ls, *minWeak))
			default:
				fmt.Print(scan.Text(target, repos, ls))
			}
			return
		}
		inv := discover.Scan(target)
		switch {
		case *prompt:
			fmt.Print(discover.Prompt(inv))
		case *asJSON:
			b, _ := json.MarshalIndent(inv, "", "  ")
			fmt.Println(string(b))
		case len(inv.Groups) == 0:
			fmt.Println("No agent harness found.")
		default:
			for _, l := range inv.Lines(1 << 30) {
				fmt.Println("- " + l)
			}
		}
	case "index":
		if len(os.Args) < 3 || os.Args[2] != "build" {
			fmt.Fprintln(os.Stderr, "usage: relai index build [-o <path>] <list-file>")
			os.Exit(2)
		}
		fs := flag.NewFlagSet("index build", flag.ExitOnError)
		out := fs.String("o", defaultIndexPath, "write the index to this file")
		var scanDirs stringList
		fs.Var(&scanDirs, "scan", "find the git repositories in this folder instead of (or in addition to) a list file; repeatable")
		scanDepth := fs.Int("depth", 1, "with --scan: how many folder levels to look down for repositories")
		_ = fs.Parse(reorderArgs(fs, os.Args[3:]))
		if fs.NArg() < 1 && len(scanDirs) == 0 {
			fmt.Fprintln(os.Stderr, "usage: relai index build [-o <path>] [--scan <folder>]... [<list-file>]")
			os.Exit(2)
		}
		var sources []string
		if fs.NArg() >= 1 {
			var err error
			sources, err = index.ReadList(fs.Arg(0))
			exitOn(err)
		}
		if len(scanDirs) > 0 {
			scanned, err := scanSources(scanDirs, *scanDepth, os.Stdout)
			exitOn(err)
			sources = append(sources, scanned...)
		}
		entries, skipped := index.Build(sources)
		for _, sk := range skipped {
			fmt.Fprintf(os.Stderr, "skipped %s: %s\n", sk.Source, sk.Reason)
		}
		if entries == nil {
			entries = []index.Entry{}
		}
		b, err := json.MarshalIndent(entries, "", "  ")
		exitOn(err)
		exitOn(os.MkdirAll(filepath.Dir(*out), 0o755))
		exitOn(os.WriteFile(*out, b, 0o644))
		for _, e := range entries {
			for _, w := range e.Warnings {
				fmt.Fprintf(os.Stderr, "warning: %s: relai.yaml %s\n", e.Source, w)
			}
		}
		fmt.Printf("wrote %s (%d repos, %d skipped)\n", *out, len(entries), len(skipped))
	case "find":
		fs := flag.NewFlagSet("find", flag.ExitOnError)
		idx := fs.String("index", defaultIndexPath, "path to the index built by `relai index build`")
		_ = fs.Parse(reorderArgs(fs, os.Args[2:]))
		if fs.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "usage: relai find [--index <path>] <capability>")
			os.Exit(2)
		}
		capability := fs.Arg(0)
		b, err := os.ReadFile(*idx)
		if err != nil {
			fmt.Printf("no index at %s — build one first: relai index build -o %s <list-file>\n", *idx, *idx)
			return
		}
		var entries []index.Entry
		exitOn(json.Unmarshal(b, &entries))
		found := index.Find(entries, capability)
		if len(found) == 0 {
			fmt.Printf("nothing in the index provides %q\n", capability)
			return
		}
		// Run inside a repo that declares `links:`: the repos it works with come first and say so.
		var me index.Manifest
		if mb, err := os.ReadFile("relai.yaml"); err == nil {
			me = index.ParseManifest(mb)
		}
		for _, e := range index.Order(found, me) {
			rel := index.Relation(e, me)
			if rel != "" {
				rel = " [" + rel + "]"
			}
			fmt.Printf("- %s (%s)%s: %s\n", e.RelaiYAML.ID, e.Source, rel, e.RelaiYAML.Summary)
			if len(e.RelaiYAML.Links) > 0 {
				fmt.Printf("    works with: %s\n", strings.Join(e.RelaiYAML.Links, ", "))
			}
		}
	case "tidy":
		fs := flag.NewFlagSet("tidy", flag.ExitOnError)
		apply := fs.Bool("apply", false, "do it (default: only show what would be done)")
		quiet := fs.Bool("quiet", false, "print only what was done, and nothing when there is nothing to do")
		limit := fs.Int("commits", 300, "how many recent commits to read")
		_ = fs.Parse(reorderArgs(fs, os.Args[2:]))
		p, err := tidy.Build(dirArg(fs), *limit)
		exitOn(err)
		if !*apply {
			if !*quiet {
				fmt.Print(p.Text())
			}
			return
		}
		r := p.Apply(dirArg(fs))
		for _, id := range r.Closed {
			fmt.Println("closed task", id)
		}
		for _, b := range r.DeletedBranches {
			fmt.Println("deleted merged branch", b)
		}
		if r.ExportRewritten {
			fmt.Println("rewrote .beads/issues.jsonl: commit it")
		}
		for _, n := range r.Notes {
			fmt.Println("note:", n)
		}
		if !*quiet && p.Deferred > 0 {
			fmt.Printf("%d task(s) have a closing commit on this branch; they will be closed once it is merged into %s.\n", p.Deferred, p.Default)
		}
		if !*quiet && len(r.Closed)+len(r.DeletedBranches) == 0 && len(r.Notes) == 0 && p.Deferred == 0 {
			fmt.Println("nothing to do.")
		}
	case "workspace":
		fs := flag.NewFlagSet("workspace", flag.ExitOnError)
		idx := fs.String("index", defaultIndexPath, "path to the index built by `relai index build`")
		max := fs.Int("max-tokens", workspace.DefaultOptions().MaxTokens, "token budget for the overview")
		_ = fs.Parse(reorderArgs(fs, os.Args[2:]))
		b, err := os.ReadFile(*idx)
		if err != nil {
			fmt.Printf("no index at %s — build one first: relai index build -o %s <list-file>\n", *idx, *idx)
			os.Exit(1)
		}
		var entries []index.Entry
		exitOn(json.Unmarshal(b, &entries))
		fmt.Print(workspace.Render(workspace.Load(entries), workspace.Options{MaxTokens: *max}))
	case "new":
		fs := flag.NewFlagSet("new", flag.ExitOnError)
		_ = fs.Parse(reorderArgs(fs, os.Args[2:]))
		if fs.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: relai new <template> <dir>")
			os.Exit(2)
		}
		tplSrc, dir := fs.Arg(0), fs.Arg(1)
		abs, err := filepath.Abs(dir)
		exitOn(err)
		r, err := scaffold.New(tplSrc, dir, scaffold.Tokens{RepoName: initrepo.Slug(filepath.Base(abs))})
		exitOn(err)
		fmt.Printf("scaffolded %s from %s\n", dir, tplSrc)
		for _, c := range r.Created {
			fmt.Println("created ", c)
		}
		for _, s := range r.Skipped {
			fmt.Println("kept    ", s)
		}
		fmt.Println()
		fmt.Println(indexNextStep(dir))
	case "version", "--version", "-v":
		fmt.Println("relai", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
}

// indexNextStep is the hint printed after a successful `relai new`, naming the command to add the
// freshly scaffolded repo to the group's cross-repo index — never run automatically.
func indexNextStep(dir string) string {
	return fmt.Sprintf("Next: add %s to your group's index — relai index build -o <path> <list-file>", dir)
}

func dirArg(fs *flag.FlagSet) string {
	if fs.NArg() > 0 {
		return fs.Arg(0)
	}
	return "."
}

func exitOn(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "relai:", err)
		os.Exit(1)
	}
}
