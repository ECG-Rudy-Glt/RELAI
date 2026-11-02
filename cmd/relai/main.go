// Command relai makes a repository agent-native: a small repo contract and a compact handoff.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
	"github.com/ECG-Rudy-Glt/RELAI/internal/initrepo"
)

var version = "dev" // set at build time: -ldflags "-X main.version=v0.1.0"

const usage = `relai — pass the baton between people and their AI agents

Usage:
  relai init [dir]       add AGENTS.md, relai.yaml and .relai/notes.md (never overwrites)
  relai handoff [dir]    print the current state of the work (≤ 2,000 tokens by default)
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
		fmt.Println("\nNext: write a one-line summary in relai.yaml, then run `relai handoff`.")
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
