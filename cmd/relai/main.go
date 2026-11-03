// Command relai makes a repository agent-native: a small repo contract and a compact handoff.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ECG-Rudy-Glt/RELAI/internal/discover"
	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
	"github.com/ECG-Rudy-Glt/RELAI/internal/initrepo"
)

// defaultIndexPath is where `relai index build` writes and `relai find` reads by default.
const defaultIndexPath = ".relai/index.json"

var version = "dev" // set at build time: -ldflags "-X main.version=v0.1.0"

const usage = `relai — pass the baton between people and their AI agents

Usage:
  relai init [dir]       add AGENTS.md, relai.yaml and .relai/notes.md (never overwrites)
  relai handoff [dir]    print the current state of the work (≤ 2,000 tokens by default)
  relai discover [dir]   list what the project already gives agents (skills, hooks, specs, docs…);
                         --prompt prints instructions for your agent to draft .relai/notes.md
  relai index build [-o <path>] <list-file>
                         read each repo path/URL in <list-file> and collect their relai.yaml
                         into an index (default: .relai/index.json)
  relai find [--index <path>] <capability>
                         search the index for repos that already provide <capability>
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
		_ = fs.Parse(os.Args[2:])
		r, err := initrepo.Init(dirArg(fs), !*noLink)
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
		_ = fs.Parse(os.Args[2:])
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
		_ = fs.Parse(os.Args[2:])
		inv := discover.Scan(dirArg(fs))
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
		_ = fs.Parse(os.Args[3:])
		if fs.NArg() < 1 {
			fmt.Fprintln(os.Stderr, "usage: relai index build [-o <path>] <list-file>")
			os.Exit(2)
		}
		sources, err := index.ReadList(fs.Arg(0))
		exitOn(err)
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
		fmt.Printf("wrote %s (%d repos, %d skipped)\n", *out, len(entries), len(skipped))
	case "find":
		fs := flag.NewFlagSet("find", flag.ExitOnError)
		idx := fs.String("index", defaultIndexPath, "path to the index built by `relai index build`")
		_ = fs.Parse(os.Args[2:])
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
		for _, e := range found {
			fmt.Printf("- %s (%s): %s\n", e.RelaiYAML.ID, e.Source, e.RelaiYAML.Summary)
		}
	case "version", "--version", "-v":
		fmt.Println("relai", version)
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
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
